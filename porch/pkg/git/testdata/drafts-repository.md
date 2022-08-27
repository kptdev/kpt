# Drafts Repository

For easier orientation, an overview of the contents of the `drafts-repository.tar`.

## Contents

The repository in `drafts-repository.tar` has the following contents in `main` branch:

```
.
├── basens
│   ├── Kptfile
│   ├── namespace.yaml
│   ├── package-context.yaml
│   └── README.md
├── empty
│   ├── Kptfile
│   └── README.md
└── istions
    ├── istions.yaml
    ├── Kptfile
    ├── package-context.yaml
    └── README.md
```

## Commits

The commit graph of the repository is:

```
| * 81d54c8aa3dd939fd36713f7074e9e1ad4f34612 (drafts/pkg-with-history/v1) Intermediate commit: eval
| * 1c3f6bf69d579b544c6e60abbba2763987d2a019 Intermediate commit: patch
| * 23371b72a0c5b5d7713030db1e9b89da402b91c4 Intermediate commit: eval
| * dc1bde5e9044f76996664978fef79d88e01e8693 Intermediate commit: init
|/
| * 487eaa0fe7652a313dcdb05790aa32034398593a (drafts/none/v1) Add none package with Kptfile marker
|/
* c7edca419782f88646f9572b0a829d686b2d91bd (HEAD -> main, tag: istions/v2) Add Istio Namespace Resource
* c93d417f1393ae5d7def978da70c42b62e645cda (tag: basens/v2) Add Base Namespace Resource
| * 032c503a3921f322850e9bd49319346e0e0b129d (drafts/bucket/v1) Add Bucket Resource
| * 1950b803f552e4c89ac17c528ec466c1a7375083 Add Bucket Package Context
| * 5ea104c29951f3c3995ae15c4a367823794bd47d Empty Bucket Package
|/  
* f8fb59f626182319ec78dd542afcce35f98811e2 (tag: istions/v1) Add Istio Namespace Package Context
* 740397bf8f594b785f6802755945bd58a5e94192 (tag: basens/v1) Add Base Namespace Package Context
* 3d21cf677b3afcca132e0b415144e83f1f2ca2a9 Empty Istio Namespace
* ca725b9cd90d5a3ee22101f0ea66f83e40217a6f Empty Base Namespace
* 7a6308e79524221d74f549bac53bf1ed771958f7 (tag: empty/v1) Empty Package

```

In addition to several published packages there are two drafts (`none` and `bucket`).
