package pkgautorun

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/google/shlex"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
)

const GeneratedDir = "generated"
const builtinMergedAnnotation = "internal.kpt.dev/generated-builtin-merged"
const (
	NativeConfigOpBackToNative = "writeBackToNativeConfig"
	NativeConfigAdaptorImage   = "gcr.io/kpt-fn-demo/configmap-generator:yuwen-v0.1"
	ConfigMapTmpFnPath         = "generated/config-map-fn-config.yaml"
)

func (r *AutoRunner) NewConfigMapGenerator(generator *fn.KubeObject, kf *kptfilev1.KptFile) (*kptfilev1.Function, error) {
	source := generator.GetMap("spec").GetSlice("source")[0]
	source.SetNestedString(NativeConfigOpBackToNative, "operation")

	for _, inclNonKrmFile := range kf.PkgAutoRun.InclNonKrmFiles {
		if inclNonKrmFile.Name == source.GetString("localFileRef") {
			source.SetNestedString(inclNonKrmFile.Path, "localFile")
		}
	}
	var fileErr error
	var f *os.File
	defer f.Close()
	os.Mkdir(GeneratedDir, os.ModePerm)
	if _, err := os.Stat(ConfigMapTmpFnPath); os.IsNotExist(err) {
		f, fileErr = os.Create(ConfigMapTmpFnPath)
	} else {
		f, fileErr = os.OpenFile(ConfigMapTmpFnPath, os.O_CREATE|os.O_WRONLY, 0660)
	}
	if fileErr != nil {
		return nil, fileErr
	}
	if _, writeErr := f.WriteString(generator.String()); writeErr != nil {
		return nil, writeErr
	}
	return &kptfilev1.Function{
		Image:      NativeConfigAdaptorImage,
		ConfigPath: ConfigMapTmpFnPath,
	}, nil
}

type AutoRunner struct {
	Ctx                context.Context
	Destination        string
	ReserveBuiltin     bool
	trackNativeConfigs fn.KubeObjects
	originGenerated    fn.KubeObjects
}

func (r *AutoRunner) RunPkgAutoPipelineWithMerge() error {
	return r.runPkgAutoPipeline(true)
}

func (r *AutoRunner) RunPkgAutoPipeline() error {
	return r.runPkgAutoPipeline(false)
}

func (r *AutoRunner) runPkgAutoPipeline(enableMerge bool) error {
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, r.Destination)
	if err != nil {
		return err
	}
	kf, err := p.Kptfile()
	if err != nil {
		return err
	}
	var nonKrmObjects fn.KubeObjects
	if kf.PkgAutoRun.InclNonKrmFiles != nil {
		nonKrmObjects, err = r.GenerateNonKrmResources(kf.PkgAutoRun.InclNonKrmFiles)
		if err != nil {
			return err
		}
	}
	var writer kio.ReaderWriter
	out := &bytes.Buffer{}
	writer = &kio.ByteReadWriter{
		Writer:                out,
		KeepReaderAnnotations: true,
		WrappingKind:          kio.ResourceListKind,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
	}
	var rl *fn.ResourceList
	var rawResourceList, output []byte

	for _, kptFunction := range kf.PkgAutoRun.BuiltInFunctions {
		configs, _ := kio.LocalPackageReader{PackagePath: filepath.Join(p.UniquePath.String(), kptFunction.ConfigPath), PreserveSeqIndent: true, WrapBareSeqNode: true}.Read()
		if len(configs) != 1 {
			return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
		}
		functionConfig := configs[0]
		rawResourceList, err = r.ReadResourceListFromFnSource(p.UniquePath.String(), functionConfig)

		if err != nil {
			return err
		}
		rl, err = fn.ParseResourceList(rawResourceList)
		if err != nil {
			return err
		}
		r.trackNativeConfigs = nonKrmObjects
		RefreshNonKrmObjects(rl, nonKrmObjects)
		var oldGenerated, newGenerated fn.KubeObjects
		if enableMerge {
			rl.Items, oldGenerated = separateGenerated(rl.Items)
			if err = r.RunFunction(writer, functionConfig, rl, kptFunction); err != nil {
				return err
			}
			if rl, err = fn.ParseResourceList(out.Bytes()); err != nil {
				return err
			}
			rl.Items, newGenerated = separateGenerated(rl.Items)
			var mergedObjects fn.KubeObjects
			mergedObjects, err = mergeGenerated(oldGenerated, r.originGenerated, newGenerated)
			if err != nil {
				return err
			}
			rl.Items = append(rl.Items, mergedObjects...)
			output, err = rl.ToYAML()
			if err != nil {
				return err
			}
			output, err = r.UpdateNonKrmAfterMerge(string(output))
			if err != nil {
				return err
			}
			output, err = r.CleanupBuiltInObjects(string(output))
			if err != nil {
				return err
			}
			return writeResourceListToLocalPackage(output, string(p.UniquePath))
		} else {
			if err = r.RunFunction(writer, functionConfig, rl, kptFunction); err != nil {
				return err
			}
			output, err = r.CleanupBuiltInObjects(out.String())
			if err != nil {
				return err
			}
			if err = r.CacheOriginFromGeneratedDir(p.UniquePath.String()); err != nil {
				return err
			}
			return writeResourceListToLocalPackage(output, string(p.UniquePath))
		}
	}
	return nil
}

func (r *AutoRunner) CacheOriginFromGeneratedDir(resolvedPath string) error {
	var buf bytes.Buffer
	p := kio.Pipeline{
		ContinueOnEmptyResult: true,
		Inputs: []kio.Reader{
			kio.LocalPackageReader{
				PackagePath:       resolvedPath,
				MatchFilesGlob:    pkg.MatchAllKRM,
				PreserveSeqIndent: true,
				WrapBareSeqNode:   true,
			},
		},
		Outputs: []kio.Writer{
			kio.ByteWriter{
				Writer:                &buf,
				KeepReaderAnnotations: true,
				WrappingKind:          kio.ResourceListKind,
				WrappingAPIVersion:    kio.ResourceListAPIVersion,
				Sort:                  true,
			},
		},
	}
	if err := p.Execute(); err != nil {
		panic(err)
	}
	rl, err := fn.ParseResourceList(buf.Bytes())
	if err != nil {
		return err
	}
	r.originGenerated = rl.Items.Where(func(o *fn.KubeObject) bool { return o.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "" })
	return nil
}

func RefreshNonKrmObjects(rl *fn.ResourceList, nonKrmObjects fn.KubeObjects) {
	rl.Items = rl.Items.WhereNot(func(o *fn.KubeObject) bool { return o.GetKind() == fn.NonKrmKind })
	rl.Items = append(rl.Items, nonKrmObjects...)
}

func writeResourceListToLocalPackage(input []byte, uniquePath string) error {
	if err := os.RemoveAll(GeneratedDir); err != nil {
		return err
	}
	pkgWriter := &kio.LocalPackageReadWriter{
		PackagePath:        uniquePath,
		MatchFilesGlob:     pkg.MatchAllKRM,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	}
	return kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBuffer(input)}},
		Outputs: []kio.Writer{pkgWriter},
	}.Execute()
}

// TODO: get Origin from the UpstreamLock
func mergeGenerated(oldObjects, originObjects, upsertedObjects fn.KubeObjects) (fn.KubeObjects, error) {
	var mergedObjects fn.KubeObjects
	for _, upserted := range upsertedObjects {

		old := oldObjects.Where(func(o *fn.KubeObject) bool {
			return o.GetAnnotation(fn.GeneratorBuiltinIdentifier) == upserted.GetAnnotation(fn.GeneratorBuiltinIdentifier) && o.GetKind() == upserted.GetKind()
		})[0]
		localRnode := old.ToRNode()
		updatedRnode := upserted.ToRNode()
		origins := originObjects.Where(func(o *fn.KubeObject) bool {
			return o.GetAnnotation(fn.GeneratorBuiltinIdentifier) == upserted.GetAnnotation(fn.GeneratorBuiltinIdentifier) && o.GetKind() == upserted.GetKind()
		})
		var originRnode *yaml.RNode
		if len(origins) > 0 {
			originRnode = origins[0].ToRNode()
		} else {
			originRnode = updatedRnode
		}
		merged, err := merge3.Merge(localRnode, originRnode, updatedRnode)
		if err != nil {
			return nil, fmt.Errorf(err.Error() + "1")
		}

		mergedKubeObject := fn.RnodeToKubeObject(merged)
		if err != nil {
			return nil, fmt.Errorf(err.Error() + "2")
		}
		mergedKubeObject.SetAnnotation(builtinMergedAnnotation, "true")
		mergedObjects = append(mergedObjects, mergedKubeObject)
	}

	return mergedObjects, nil
}

func separateGenerated(items fn.KubeObjects) (fn.KubeObjects, fn.KubeObjects) {
	var nonGeneratedObjects fn.KubeObjects
	var generatedObjects fn.KubeObjects
	for _, object := range items {
		switch true {
		case object.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "":
			generatedObjects = append(generatedObjects, object)
		default:
			nonGeneratedObjects = append(nonGeneratedObjects, object)
		}
	}
	return nonGeneratedObjects, generatedObjects
}

func getFunctionSpec(image, exec string) (*runtimeutil.FunctionSpec, error) {
	fn := &runtimeutil.FunctionSpec{}
	if image != "" {
		fn.Container.Image = image
	} else if exec != "" {
		s, err := shlex.Split(exec)
		if err != nil {
			return nil, fmt.Errorf("exec command %q must be valid: %w", exec, err)
		}
		if len(s) > 0 {
			fn.Exec.Path = s[0]
		}
	}
	return fn, nil
}

func (r *AutoRunner) RunFunction(writer kio.Writer, functionConfig *yaml.RNode, inputResourceList *fn.ResourceList, function kptfilev1.Function) error {
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, r.Destination)
	output := &bytes.Buffer{}
	content, _ := inputResourceList.ToYAML()
	input := bytes.NewReader(content)
	fnSpec, err := getFunctionSpec(function.Image, function.Exec)
	if err != nil {
		return err
	}
	run := runfn.RunFns{
		Ctx:                   r.Ctx,
		Input:                 input,
		Output:                output,
		ImagePullPolicy:       fnruntime.IfNotPresentPull,
		AsCurrentUser:         false,
		ContinueOnEmptyResult: true,
		Function:              fnSpec,
		OriginalExec:          function.Exec,
		Path:                  p.UniquePath.String(),
		FnConfig:              functionConfig,
	}
	if err = runner.HandleError(r.Ctx, run.Execute()); err != nil {
		return err
	}
	rl, _ := fn.ParseResourceList(output.Bytes())
	for _, o := range rl.Items {
		if o.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "" || o.GetAnnotation(fn.GeneratorIdentifier) != "" || o.GetKind() == fn.NonKrmKind {
			curFilePath := o.GetAnnotation(fn.PathAnnotation)
			newFilePath := filepath.Join(GeneratedDir, filepath.Base(curFilePath))
			o.SetAnnotation(fn.PathAnnotation, newFilePath)
			o.SetAnnotation(kioutil.LegacyPathAnnotation, newFilePath)
			o.SetAnnotation(kioutil.LegacyIndexAnnotation, o.GetAnnotation(fn.IndexAnnotation))
		}
	}

	newOutput, err := rl.ToYAML()
	if err != nil {
		return err
	}
	return kio.Pipeline{
		Inputs: []kio.Reader{&kio.ByteReader{
			Reader:             bytes.NewBuffer(newOutput),
			PreserveSeqIndent:  true,
			WrapBareSeqNode:    true,
			WrappingKind:       kio.ResourceListKind,
			WrappingAPIVersion: kio.ResourceListAPIVersion,
		}},
		Outputs: []kio.Writer{writer},
	}.Execute()
}

func IsInternalGenerator(o *fn.KubeObject) bool {
	if o.GetKind() != "ConfigMapGenerator" {
		return false
	}
	sources := o.GetMap("spec").GetSlice("source")
	for _, source := range sources {

		val, _, _ := source.NestedString("operation")
		if val == NativeConfigOpBackToNative {
			return true
		}
	}
	return false
}
func (r *AutoRunner) CleanupBuiltInObjects(data string) ([]byte, error) {
	p, _ := pkg.New(filesys.FileSystemOrOnDisk{}, r.Destination)
	os.Remove(filepath.Join(p.UniquePath.String(), ConfigMapTmpFnPath))
	rl, err := fn.ParseResourceList([]byte(data))
	if err != nil {
		return nil, err
	}
	var newItems fn.KubeObjects
	for _, item := range rl.Items {
		if item.GetKind() == fn.NonKrmKind {
			continue
		}
		if !r.ReserveBuiltin && item.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "" {
			continue
		}

		if item.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "" || item.GetAnnotation(fn.GeneratorIdentifier) != "" {
			curFilePath := item.GetAnnotation(fn.PathAnnotation)
			newFilePath := filepath.Join(GeneratedDir, filepath.Base(curFilePath))
			item.SetAnnotation(fn.PathAnnotation, newFilePath)
			item.SetAnnotation(kioutil.LegacyPathAnnotation, newFilePath)
			item.SetAnnotation(kioutil.LegacyIndexAnnotation, item.GetAnnotation(fn.IndexAnnotation))
		}
		newItems = append(newItems, item)
	}
	rl.Items = newItems
	return rl.ToYAML()
}

func (r *AutoRunner) UpdateNonKrmAfterMerge(data string) ([]byte, error) {
	rl, err := fn.ParseResourceList([]byte(data))
	if err != nil {
		return nil, err
	}
	// should have single builtin merge anno
	sourceOfTruthCanonicalFn := func(o *fn.KubeObject) bool {
		return o.GetAnnotation(builtinMergedAnnotation) != ""
	}
	sotCanonicalObjects := rl.Items.Where(sourceOfTruthCanonicalFn)
	if len(sotCanonicalObjects) == 0 {
		sotCanonicalObjects = rl.Items.Where(func(o *fn.KubeObject) bool {
			return o.GetAnnotation(fn.GeneratorBuiltinIdentifier) != ""
		})
	}
	if len(sotCanonicalObjects) > 1 {
		sotCanonicalObjects = []*fn.KubeObject{sotCanonicalObjects[len(sotCanonicalObjects)-1]}
	}
	for _, obj := range sotCanonicalObjects {
		// Get the matching Generator from annotation
		generatorFnConfigResourceID := obj.GetAnnotation(fn.GeneratorBuiltinIdentifier)
		if generatorFnConfigResourceID == "" {
			return nil, fmt.Errorf("missing generator builtin annotation %v", obj.String())
		}
		generatorFnConfigs := rl.Items.Where(func(o *fn.KubeObject) bool {
			return o.GetKind() == "ConfigMapGenerator"
		})
		if len(generatorFnConfigs) == 0 {
			return nil, fmt.Errorf("missing generator function config")
		}
		// WriteToNonKrmFromCanonical removes old nonKRM object, so WriteNonKrmToNativeFile can just write all native config.
		if err = r.WriteToNonKrmFromCanonical(generatorFnConfigs[0], rl); err != nil {
			return nil, err
		}
		if err = r.WriteNonKrmToNativeFile(rl); err != nil {
			return nil, err // should have single builtin merge
		}
	}
	return rl.ToYAML()
}

// WriteNonKrmToNativeFile write non KRM object back to the native file and remove the non KRM object from the resourcelist.
func (r *AutoRunner) WriteNonKrmToNativeFile(rl *fn.ResourceList) error {
	rawConfigMaps := rl.Items.Where(func(o *fn.KubeObject) bool {
		return o.GetKind() == "ConfigMap" && o.GetAnnotation(fn.GeneratorIdentifier) != ""
	})
	if len(rawConfigMaps) != 1 {
		return fmt.Errorf("expect a single raw ConfigMap. got " + rawConfigMaps.String())
	}
	rawFileAndContent, found, err := rawConfigMaps[0].NestedStringMap("data")
	if !found || err != nil {
		return fmt.Errorf("no data from SOT canonical ConfigMap")
	}
	var f *os.File
	defer f.Close()
	for file, content := range rawFileAndContent {
		var fileErr error
		if _, err := os.Stat(file); os.IsNotExist(err) {
			f, fileErr = os.Create(file)
		} else {
			f, fileErr = os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0660)
		}
		if fileErr != nil {
			return fileErr
		}
		if _, writeErr := f.WriteString(content); writeErr != nil {
			return writeErr
		}
		pr := printer.FromContextOrDie(r.Ctx)
		pr.Printf("Successfully update " + file + "\n")
	}
	return nil
}

func (r *AutoRunner) WriteToNonKrmFromCanonical(fnConfig *fn.KubeObject, rl *fn.ResourceList) error {
	out := &bytes.Buffer{}
	writer := kio.ByteWriter{
		Writer:                out,
		KeepReaderAnnotations: true,
		WrappingKind:          kio.ResourceListKind,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		Sort:                  true,
	}
	// Get filename from Kptfile pkgAutoRun
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, r.Destination)
	if err != nil {
		return err
	}
	kf, err := p.Kptfile()
	if err != nil {
		return err
	}
	pr := printer.FromContextOrDie(r.Ctx)
	// fnConfig should be ConfigMapGenerator
	function, err := r.NewConfigMapGenerator(fnConfig, kf)
	if err != nil {
		return err
	}

	rl.Items = rl.Items.WhereNot(fn.IsNonKrmObject)
	configs, _ := kio.LocalPackageReader{PackagePath: filepath.Join(p.UniquePath.String(), ConfigMapTmpFnPath), PreserveSeqIndent: true, WrapBareSeqNode: true}.Read()
	if len(configs) == 0 {
		return fmt.Errorf("unable to write configMapGenerator fn-config to " + ConfigMapTmpFnPath)
	}
	if err := r.RunFunction(writer, configs[0], rl, *function); err != nil {
		pr.Printf("fail run function " + rl.Items.String() + "\n\n")

		return err
	}
	newrl, err := fn.ParseResourceList(out.Bytes())
	if err != nil {
		return err
	}
	rl.Items = newrl.Items
	for _, o := range rl.Items {
		if o.GetId().String() == fnConfig.GetId().String() {
			sources := o.GetMap("spec").GetSlice("source")
			for _, source := range sources {
				source.RemoveNestedField("operation")
				source.RemoveNestedField("localFile")
			}
		}
	}
	rl.Results = newrl.Results
	return nil
}

func (r *AutoRunner) GenerateNonKrmResources(includeNonKrmFiels []kptfilev1.LocalFile) (fn.KubeObjects, error) {
	var nonKrmObjects []*fn.KubeObject
	for _, localNonKrmFile := range includeNonKrmFiels {
		content, err := ioutil.ReadFile(filepath.Join(r.Destination, localNonKrmFile.Path))
		if err != nil {
			return nil, err
		}
		newNonKrmFile := fn.NewNonKrmResource()
		obj, err := fn.NewFromTypedObject(newNonKrmFile)
		if err != nil {
			return nil, err
		}
		obj.SetName(localNonKrmFile.Name)
		obj.SetNestedString(string(content), "spec", "content")
		obj.SetNestedString(filepath.Base(localNonKrmFile.Path), "spec", "filename")
		nonKrmObjects = append(nonKrmObjects, obj)
	}
	return nonKrmObjects, nil
}

func (r *AutoRunner) ReadResourceListFromFnSource(resolvedPath string, fnConfig *yaml.RNode) ([]byte, error) {
	var inputs []kio.Reader
	inputs = append(inputs, kio.LocalPackageReader{
		PackagePath:        resolvedPath,
		MatchFilesGlob:     pkg.MatchAllKRM,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	})
	var outputs []kio.Writer
	actual := &bytes.Buffer{}
	outputs = append(outputs, kio.ByteWriter{
		Writer:             actual,
		FunctionConfig:     fnConfig,
		WrappingKind:       kio.ResourceListKind,
		WrappingAPIVersion: kio.ResourceListAPIVersion,
	})
	err := kio.Pipeline{Inputs: inputs, Outputs: outputs}.Execute()
	if err != nil {
		return nil, err
	}
	return actual.Bytes(), err
}

func (r *AutoRunner) DeleteNativeConfig() error {
	for _, o := range r.trackNativeConfigs {
		fpath := filepath.Join(r.Destination, o.GetMap("spec").GetString("filename"))
		if err := os.Remove(fpath); err != nil {
			return err
		}
	}
	return nil
}
