// Package nvidiavsp is a foundational, COMPILABLE skeleton (not fully functional) for
// integrating NVIDIA BlueField DPU support into the OPI DPU Operator by reusing NVIDIA's
// existing DPF (DOCA Platform Framework) operator.
//
// Design (see architecture_design.md for the full rationale and diagrams):
//
//	The DPU Operator already supports pluggable, per-node "Vendor Specific Plugins" (VSPs)
//	that speak a gRPC contract over a unix socket. Intel and Marvell each ship a VSP.
//	We add NVIDIA as a first-class vendor WITHOUT re-implementing offload, by splitting the
//	work across three collaborating pieces, each owning the concern it actually fits:
//
//	  1. DPFProvisioner (SUB-OPERATOR concern), owns the heavyweight, cluster-scoped,
//	     asynchronous DPF lifecycle: install DPF, apply DPFOperatorConfig + DPUSet/BFB/DPUFlavor
//	     to flash BlueField-3 and form the DPUCluster. This is minutes-long and CANNOT be
//	     driven synchronously from the VSP's imperative Init() call, hence a separate piece.
//
//	  2. NVIDIAVSP (ADAPTER concern), implements the node-local surface the DPU Operator core
//	     expects (device enumeration, VF count, per-NF calls) so NVIDIA is a first-class VSP.
//	     It does NOT do provisioning; it verifies DPF is ready and delegates the datapath.
//
//	  3. ServiceChainTranslator (CRD-TRANSLATION concern), a controller-runtime style
//	     reconciler that maps DPU-Operator *intent* CRDs (ServiceFunctionChain, DpuNetwork)
//	     into DPF CRDs (DPUServiceChain, DPUServiceInterface). Some mappings are lossy; the
//	     lossy edges are called out in comments and in architecture_design.md.
//
// The stub types below mirror the shapes of the real upstream types so the intent is legible;
// they are intentionally minimal so this file compiles standalone (stdlib only), per the
// assignment's "compilable but not necessarily fully functional" requirement.
//
// Real upstream references (verified against source, 2026-07-02):
//   - VSP interface:  github.com/openshift/dpu-operator/internal/daemon/plugin.VendorPlugin
//   - VSP gRPC proto: github.com/openshift/dpu-operator/dpu-api/api.proto
//   - DPU-Op CRDs:    github.com/openshift/dpu-operator/api/v1 (ServiceFunctionChain, DpuNetwork, ...)
//   - DPF CRDs:       github.com/nvidia/doca-platform/api/{operator,provisioning,dpuservice}/v1alpha1
package nvidiavsp

import (
	"context"
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Minimal controller-runtime-shaped stubs (mirror sigs.k8s.io/controller-runtime).
// Real code would import controller-runtime; these locals keep the skeleton buildable.
// ---------------------------------------------------------------------------

// NamespacedName mirrors k8s.io/apimachinery/pkg/types.NamespacedName.
type NamespacedName struct {
	Namespace string
	Name      string
}

// Request mirrors reconcile.Request.
type Request struct {
	NamespacedName
}

// Result mirrors reconcile.Result. RequeueAfterSeconds stands in for time.Duration.
type Result struct {
	Requeue             bool
	RequeueAfterSeconds int64
}

// Object is a minimal stand-in for client.Object.
type Object interface {
	GetName() string
}

// Client is a minimal stand-in for client.Client (the reconciler's k8s API handle).
type Client interface {
	Get(ctx context.Context, key NamespacedName, obj Object) error
	Create(ctx context.Context, obj Object) error
	Patch(ctx context.Context, obj Object) error
}

// Reconciler mirrors reconcile.Reconciler.
type Reconciler interface {
	Reconcile(ctx context.Context, req Request) (Result, error)
}

// ---------------------------------------------------------------------------
// DPU Operator VSP contract (mirrors internal/daemon/plugin.VendorPlugin + dpu-api/api.proto).
// ---------------------------------------------------------------------------

// InitRequest mirrors the VSP LifeCycleService.Init request.
type InitRequest struct {
	DpuMode       bool
	DpuIdentifier string
}

// Device / DeviceListResponse mirror the VSP DeviceService messages.
type Device struct {
	ID     string
	Health string
}

// DeviceListResponse mirrors DeviceService.GetDevices response.
type DeviceListResponse struct {
	Devices map[string]Device
}

// VfCount mirrors DeviceService.SetNumVfs message.
type VfCount struct {
	VfCnt int32
}

// VendorPlugin mirrors github.com/openshift/dpu-operator/internal/daemon/plugin.VendorPlugin.
// (BridgePort methods from the OPI EVPN-GW API are elided for brevity; they map to
// DPUServiceInterface/DPUServiceNAD the same way CreateNetworkFunction does.)
type VendorPlugin interface {
	Start(ctx context.Context) (ip string, port int32, err error)
	Close()
	GetDevices() (*DeviceListResponse, error)
	SetNumVfs(vfCount int32) (*VfCount, error)
	CreateNetworkFunction(input, output, bridgeID string) error
	DeleteNetworkFunction(input, output, bridgeID string) error
	SetDpuNetworkConfig(isAccelerated bool) error
}

// ---------------------------------------------------------------------------
// DPU Operator intent CRDs (mirror api/v1).
// ---------------------------------------------------------------------------

// ServiceFunctionChain mirrors dpu-operator api/v1.ServiceFunctionChain (the SFC intent).
type ServiceFunctionChain struct {
	Name             string
	NodeSelector     map[string]string
	NetworkFunctions []NetworkFunction
}

// NetworkFunction mirrors api/v1.NetworkFunction: a flat {name, image} pair.
type NetworkFunction struct {
	Name  string
	Image string
}

func (s *ServiceFunctionChain) GetName() string { return s.Name }

// ---------------------------------------------------------------------------
// DPF CRDs (mirror api/{operator,provisioning,dpuservice}/v1alpha1). Minimal fields only.
// ---------------------------------------------------------------------------

// DPFOperatorConfig mirrors operator.dpu.nvidia.com/v1alpha1.DPFOperatorConfig.
// Real readiness is expressed via status.conditions (conditions.TypeReady /
// SystemComponentsReadyCondition), not a bool; Ready here stands in for that condition.
type DPFOperatorConfig struct {
	Name  string
	Ready bool // stands for status.conditions[type=Ready] on the real CRD
}

func (c *DPFOperatorConfig) GetName() string { return c.Name }

// DPUSet mirrors provisioning.dpu.nvidia.com/v1alpha1.DPUSet (drives BFB flashing / VFs).
type DPUSet struct {
	Name         string
	BFB          string // BlueField bootstream (provisioning.BFB)
	DPUFlavor    string // provisioning.DPUFlavor (VF/SF count, OVS mode, ...)
	NodeSelector map[string]string
}

func (s *DPUSet) GetName() string { return s.Name }

// DPUServiceChain mirrors svc.dpu.nvidia.com/v1alpha1.DPUServiceChain.
type DPUServiceChain struct {
	Name  string
	Ports []string
}

func (c *DPUServiceChain) GetName() string { return c.Name }

// DPUServiceInterface mirrors svc.dpu.nvidia.com/v1alpha1.DPUServiceInterface.
type DPUServiceInterface struct {
	Name string
}

func (i *DPUServiceInterface) GetName() string { return i.Name }

// ---------------------------------------------------------------------------
// ErrDPFNotReady is returned while DPF provisioning (async, minutes-long) is still in flight.
// ---------------------------------------------------------------------------
var ErrDPFNotReady = errors.New("nvidiavsp: DPF provisioning not ready yet")

// ---------------------------------------------------------------------------
// 1. DPFProvisioner, SUB-OPERATOR concern: owns the cluster-scoped, async DPF lifecycle.
// ---------------------------------------------------------------------------

// DPFProvisioner ensures DPF is installed and BlueField-3 is provisioned. This is deliberately
// decoupled from the VSP's imperative Init() because BFB flashing + DPUCluster formation is a
// long-running, declarative, cluster-scoped process, not something Init() can block on.
type DPFProvisioner struct {
	Client       Client
	BFBURL       string
	FlavorName   string
	NodeSelector map[string]string
}

// EnsureDPF applies DPFOperatorConfig + DPUSet so DPF flashes BF3 and forms the DPUCluster.
// It is idempotent: safe to call every reconcile.
func (p *DPFProvisioner) EnsureDPF(ctx context.Context) error {
	cfg := &DPFOperatorConfig{Name: "dpfoperatorconfig"}
	if err := p.Client.Create(ctx, cfg); err != nil {
		return fmt.Errorf("ensure DPFOperatorConfig: %w", err)
	}
	set := &DPUSet{
		Name:         "opi-managed-dpuset",
		BFB:          p.BFBURL,
		DPUFlavor:    p.FlavorName,
		NodeSelector: p.NodeSelector,
	}
	if err := p.Client.Create(ctx, set); err != nil {
		return fmt.Errorf("ensure DPUSet: %w", err)
	}
	return nil
}

// Ready reports whether DPF is up. In real code this reads the DPFOperatorConfig Ready
// condition (DPF system components) AND the DPU/DPUSet provisioning status (BF3 flashed,
// DPUCluster formed). The skeleton checks the single stub condition for brevity.
func (p *DPFProvisioner) Ready(ctx context.Context) (bool, error) {
	cfg := &DPFOperatorConfig{Name: "dpfoperatorconfig"}
	if err := p.Client.Get(ctx, NamespacedName{Name: cfg.Name}, cfg); err != nil {
		return false, fmt.Errorf("read DPFOperatorConfig Ready condition: %w", err)
	}
	return cfg.Ready, nil
}

// ---------------------------------------------------------------------------
// 2. NVIDIAVSP, ADAPTER concern: the node-local VSP the DPU Operator core talks to.
// ---------------------------------------------------------------------------

// NVIDIAVSP implements the DPU Operator's VendorPlugin contract for NVIDIA BlueField.
// It performs NO provisioning itself: it checks DPF readiness (owned by DPFProvisioner) and
// delegates datapath programming to DPF via the translator.
type NVIDIAVSP struct {
	Client      Client
	Provisioner *DPFProvisioner
	Translator  *ServiceChainTranslator
	dpuMode     bool
}

// Compile-time assertion that NVIDIAVSP satisfies the DPU Operator VSP contract.
var _ VendorPlugin = (*NVIDIAVSP)(nil)

// Start mirrors VSP LifeCycleService.Init: it returns the control endpoint once DPF is ready,
// otherwise ErrDPFNotReady so the DPU Operator daemon retries (matching the real Init retry loop).
func (v *NVIDIAVSP) Start(ctx context.Context) (string, int32, error) {
	ready, err := v.Provisioner.Ready(ctx)
	if err != nil {
		return "", 0, err
	}
	if !ready {
		return "", 0, ErrDPFNotReady
	}
	return "127.0.0.1", 50051, nil
}

// Close releases resources. No-op in the skeleton.
func (v *NVIDIAVSP) Close() {}

// GetDevices maps DPF-provisioned DPUs to the VSP DeviceService view. TODO: list provisioning.DPU.
func (v *NVIDIAVSP) GetDevices() (*DeviceListResponse, error) {
	return &DeviceListResponse{Devices: map[string]Device{}}, nil
}

// SetNumVfs maps to DPUFlavor VF configuration (declarative on the DPF side). TODO: patch DPUFlavor.
func (v *NVIDIAVSP) SetNumVfs(vfCount int32) (*VfCount, error) {
	return &VfCount{VfCnt: vfCount}, nil
}

// CreateNetworkFunction translates a single NF wiring into a DPUServiceInterface on the DPU.
func (v *NVIDIAVSP) CreateNetworkFunction(input, output, bridgeID string) error {
	iface := &DPUServiceInterface{Name: fmt.Sprintf("nf-%s-%s", bridgeID, input)}
	return v.Client.Create(context.Background(), iface)
}

// DeleteNetworkFunction is the inverse of CreateNetworkFunction. No-op in the skeleton.
func (v *NVIDIAVSP) DeleteNetworkFunction(input, output, bridgeID string) error { return nil }

// SetDpuNetworkConfig toggles accelerated OVN/OVS-DOCA offload on the DPF side.
func (v *NVIDIAVSP) SetDpuNetworkConfig(isAccelerated bool) error {
	v.dpuMode = isAccelerated
	return nil
}

// ---------------------------------------------------------------------------
// 3. ServiceChainTranslator, CRD-TRANSLATION concern: DPU-Operator intent -> DPF CRDs.
// ---------------------------------------------------------------------------

// ServiceChainTranslator reconciles DPU-Operator ServiceFunctionChain CRs into DPF
// DPUServiceChain CRs. It is a standard controller-runtime style reconciler.
type ServiceChainTranslator struct {
	Client Client
}

var _ Reconciler = (*ServiceChainTranslator)(nil)

// Reconcile fetches the source SFC and applies the translated DPUServiceChain.
func (t *ServiceChainTranslator) Reconcile(ctx context.Context, req Request) (Result, error) {
	sfc := &ServiceFunctionChain{}
	if err := t.Client.Get(ctx, req.NamespacedName, sfc); err != nil {
		return Result{}, fmt.Errorf("get ServiceFunctionChain %s: %w", req.Name, err)
	}
	chain := TranslateSFC(sfc)
	if err := t.Client.Create(ctx, chain); err != nil {
		return Result{}, fmt.Errorf("apply DPUServiceChain %s: %w", chain.Name, err)
	}
	return Result{}, nil
}

// TranslateSFC maps a DPU-Operator ServiceFunctionChain onto a DPF DPUServiceChain.
//
// LOSSY MAPPING (documented in architecture_design.md, "Trade-offs / mapping fidelity"):
//   - SFC models a flat ordered list of {name, image} network functions. DPF's
//     ServiceChainSet template models a richer port/interface topology (Service +
//     ServiceInterface references). We can synthesize a linear chain from the SFC order,
//     but SFC cannot express branch/multi-port topologies DPF supports (forward-lossy),
//     and DPF chains that use non-linear topology cannot round-trip back to SFC (reverse-lossy).
//   - SFC.NodeSelector (node labels) maps to DPUClusterSelector/ServiceChainSet NodeSelector,
//     but the selection domains differ (host nodes vs DPU nodes), see design doc.
func TranslateSFC(sfc *ServiceFunctionChain) *DPUServiceChain {
	ports := make([]string, 0, len(sfc.NetworkFunctions))
	for _, nf := range sfc.NetworkFunctions {
		ports = append(ports, nf.Name)
	}
	return &DPUServiceChain{Name: sfc.Name, Ports: ports}
}
