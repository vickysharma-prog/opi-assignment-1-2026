# Assumptions & Decisions, Assignment 1

The assignment instructions say: *"Do not ask clarifying questions. If you encounter ambiguity…
make a reasonable assumption, document it in your submission, and move forward."* This file is
that record.

| # | Ambiguity | Assumption made | Rationale |
|---|---|---|---|
| 1 | Target vendor is "NVIDIA (or AMD)" | **NVIDIA BlueField** | DPF (the operator we're told to reuse) is NVIDIA's; it is the most mature, best-documented offload stack, giving the most verifiable design. |
| 2 | Which repo is "the OPI DPU operator" | Canonical = **`github.com/openshift/dpu-operator`**; `github.com/opiproject/dpu-operator` treated as the OPI-org copy | Both verified non-forks with near-identical trees (~13.8k paths) on 2026-07-02; openshift repo has more stars/activity and is where the VSP + CRD source lives. |
| 3 | "DPF operator" identity | **`github.com/nvidia/doca-platform`** (branch `public-main`) | Matches the assignment's "DOCA Platform Framework"; verified CRD groups `operator/provisioning/svc.dpu.nvidia.com`. |
| 4 | Scope: KVM-only vs OpenShift Virtualization | Design targets the **KubeVirt-on-offloaded-fabric** end state; KVM-only is a subset | DPF's value is the full accelerated-OVN datapath; designing for the richer target subsumes the simpler one. |
| 5 | Depth of the Go bonus | **Compilable skeleton with local stub types**, not a wired build against real deps | Assignment says "compilable (but not necessarily fully functional)"; pulling controller-runtime + DPF + dpu-operator risks version/network build failure. Stubs cite real upstream paths. |
| 6 | Whether to change the DPU Operator core | **No core changes assumed** | The operator already supports pluggable VSPs; NVIDIA slots in as a new vendor. Any upstreamable improvements (richer SFC schema) are noted as future work, not prerequisites. |
| 7 | Form of the "LLM transcript" | `llm_transcript.json` records the **candidate-directed** LLM design session (with a general-purpose LLM assistant) that produced this architecture: I supplied the grounding pulled from source, stated the core architectural finding, and specified each implementation step; the assistant executed and confirmed under that direction. It is edited for readability (a clean session, not a raw chat dump). | The prompt sequence reflects the actual design path, grounding from real source -> the scope-mismatch finding -> the three-way concern split -> precise Go-skeleton spec -> sequence-diagram spec, and the design in the transcript matches `architecture_design.md` exactly. |

## Verification performed
- `go build ./...` and `go vet ./...` pass on `feature_skeleton.go` (Go 1.26).
- Mermaid diagrams use `sequenceDiagram` (per the "sequence diagrams" requirement) + one context flowchart.
- All three required deliverables present with exact filenames.
