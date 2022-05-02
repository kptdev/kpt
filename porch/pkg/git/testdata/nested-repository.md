# Nested Repository

For easier orientation, an overview of the contents of the `nested-repository.tar`.

## Contents

The repository in `nested-repository.tar` has the following contents
in `main` branch:

```
.
├── catalog
│   ├── empty
│   │   ├── Kptfile
│   │   └── README.md
│   ├── gcp
│   │   └── bucket
│   │       ├── bucket.yaml
│   │       ├── Kptfile
│   │       ├── package-context.yaml
│   │       └── README.md
│   └── namespace
│       ├── basens
│       │   ├── Kptfile
│       │   ├── namespace.yaml
│       │   ├── package-context.yaml
│       │   └── README.md
│       └── istions
│           ├── Kptfile
│           ├── namespace.yaml
│           ├── package-context.yaml
│           └── README.md
└── sample
    ├── Kptfile
    ├── package-context.yaml
    └── README.md
```

## Commits

The commit graph of the repository is:

```
* 1149bc62eca4a4a28b40695bcf44c22a4d28bc17 (drafts/catalog/gcp/cloud-sql/v1) Cloud SQL Package
| * d671169ffac0c7587b4dc41c667276f14c023fd1 (drafts/catalog/gcp/spanner/v1) Spanner Package
| | * e495fc033b38c5873b4575f21ded28e923744d04 (drafts/catalog/gcp/bucket/v2) Enable Bucket Versioning
| |/  
| * 1e155ee719634981881bcb696530e355fa9c9aba (HEAD -> main, tag: sample/v2) Sample Package Context
|/  
* 27c4e150a4a6b19ca4f54e3ba68779ebf3a04845 (tag: catalog/gcp/bucket/v1) Bucket Package
* 46ddff37bea3fda36a1b64b88539d89064d7f163 (tag: catalog/namespace/basens/v3) Base Namespace Resource
* 381c4a21013c5cd7d4304e069e9f7135adeddee3 (tag: catalog/namespace/istions/v3) Istio Namespace Resource
* c7b38567d2d12dd1848ed0a6bdc55a121bbe2fa2 (tag: catalog/namespace/istions/v2) Istio Namespacd Package Context
* e7ae90895740d60b01101293311f64c2d16e64eb (tag: catalog/namespace/basens/v2) Base Namespace Package Context
* 0c94de6fd7f46e8a2d1fe1a60d90a759b54cae94 (tag: catalog/namespace/istions/v1) Istio Namespace
* f006a6654ef6d06010d1460038b993934151da08 (tag: catalog/namespace/basens/v1) Base Namespace
* b8d4e1b2c3a87dd98c55f8991391c740a550a0bf (tag: catalog/empty/v1) Empty Package
* caac51094e012cf41b4dcdb14a269b8145bf0fbf (tag: sample/v1) Sample Package
* 0f890879a10df9aaa9b263035eea955335451804 Empty Commit

```
