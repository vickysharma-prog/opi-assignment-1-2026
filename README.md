# Assignment 1, LLM-Assisted Architecture Design for OPI DPU Operator

**Candidate:** Vicky Sharma · github.com/vickysharma-prog
**Assignment:** https://github.com/sknrao/opi-assignment-1-2026

## What this is
An LLM-assisted software-architecture design that adds **NVIDIA BlueField DPU support** to the
**OPI DPU Operator** by **reusing NVIDIA's existing DPF (DOCA Platform Framework) operator**, 
so NVIDIA becomes a first-class vendor in the OPI ecosystem without re-implementing offload.

## Deliverables (exact filenames required by the assignment)
| File | What it is |
|---|---|
| `architecture_design.md` | Final architecture: current-state analysis (grounded in real source), the core impedance finding, the 3-component design, **Mermaid sequence diagrams**, trade-off analysis, lossy-mapping table. |
| `llm_transcript.json` | The **candidate-directed** design session (LLM) that produced the design, as `[{"role","content"}, …]`: I supply the grounding, the core insight, and each implementation directive; the assistant executes/confirms. Edited for readability; the design matches this doc exactly (see `ASSUMPTIONS.md` #7). |
| `feature_skeleton.go` | Compilable Go skeleton of the three components (adapter VSP + sub-operator provisioner + CRD-translation reconciler). |

Supporting: `ASSUMPTIONS.md` (documented assumptions per the "assume & move on" rule), `go.mod`.

## The design in one paragraph
The DPU Operator integrates vendors via a **per-node, imperative gRPC "Vendor Specific Plugin"
(VSP)** seam (Intel and Marvell each ship one). DPF, in contrast, is **cluster-scoped, declarative,
and asynchronous** (flashing BF3 is minutes long). So a *pure* VSP adapter cannot maximize DPF
reuse, it would bottleneck DPF through a synchronous straw. The design therefore **splits by
concern**: a **sub-operator** owns the async DPF provisioning lifecycle; a thin **VSP adapter**
covers the node-local device/VF surface (keeping NVIDIA first-class); a **CRD-translation
reconciler** maps DPU-Operator intent (`ServiceFunctionChain`) to DPF CRDs (`DPUServiceChain`),
with lossy edges made explicit. DPF does all the heavy lifting.

## Verify
```bash
go build ./...   # compiles (Go 1.23+); go vet ./... also clean
python -c "import json;json.load(open('llm_transcript.json'))"   # valid JSON
```
Render `architecture_design.md` in any Mermaid-aware viewer (GitHub renders it inline).

## Grounding
The design was verified against real source on 2026-07-02: the `VendorPlugin` interface and
`dpu-api/api.proto` in `openshift/dpu-operator`, its `api/v1` CRDs, and the CRD groups in
`nvidia/doca-platform`. Nothing here is invented.
