// Package ovf extracts OVF/OVA package metadata by running
// `ovftool --machineOutput <source>` in probe mode (no target) and parsing
// its structured output, rather than hand-rolling an OVF/tar/manifest
// parser — ovftool already handles both .ovf and .ova (and even remote
// URLs) authoritatively, so we lean on it for descriptor extraction the
// same way we lean on it for the deploy itself.
package ovf

// probeResult mirrors the <probeResult> element ovftool emits under the
// PROBE section of --machineOutput. Field order follows the source.
type probeResult struct {
	VirtualApp  bool        `xml:"virtualApp"`
	ProductInfo productInfo `xml:"productInfo"`
	Annotation  string      `xml:"annotation"`
	Sizes       probeSizes  `xml:"sizes"`
	Networks    []probeNet  `xml:"networks>network"`
	VMs         []probeVM   `xml:"vms>vm"`
	Properties  []probeProp `xml:"properties>property"`
	References  []string    `xml:"References>File"`
}

type productInfo struct {
	Name        string `xml:"name"`
	ProductURL  string `xml:"productUrl"`
	Version     string `xml:"version"`
	FullVersion string `xml:"fullVersion"`
	Vendor      string `xml:"vendor"`
	VendorURL   string `xml:"vendorUrl"`
}

// probeSizes holds byte counts as strings because the "sparse" size is
// often the literal text "Unknown" rather than a number.
type probeSizes struct {
	Download              string `xml:"download"`
	Flat                  string `xml:"flat"`
	Sparse                string `xml:"sparse"`
	HasVariableSizedDisks bool   `xml:"hasVariableSizedDisks"`
}

type probeNet struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	FenceMode   string `xml:"fenceMode"`
}

type probeVM struct {
	ID               string            `xml:"id,attr"`
	Name             string            `xml:"name"`
	OSIDs            []string          `xml:"osIds>osId"`
	VirtualHardwares []virtualHardware `xml:"virtualHardwares>virtualHardware"`
}

type virtualHardware struct {
	Families       []string    `xml:"families>family"`
	Disks          []probeDisk `xml:"disks>disk"`
	NumberOfCpus   int         `xml:"numberOfCpus"`
	CoresPerSocket string      `xml:"coresPerSocket"`
	MemoryBytes    int64       `xml:"memory"`
}

type probeDisk struct {
	Index        int      `xml:"index"`
	InstanceID   string   `xml:"instanceId"`
	CapacityByte int64    `xml:"capacity"`
	HostResource string   `xml:"hostResource"`
	Controllers  []string `xml:"diskControllers>diskController"`
}

type probeProp struct {
	Key         string `xml:"key"`
	ClassID     string `xml:"classId"`
	InstanceID  string `xml:"instanceId"`
	Category    string `xml:"category"`
	Label       string `xml:"label"`
	Type        string `xml:"type"`
	Description string `xml:"description"`
	Value       string `xml:"value"`
}

// probeWarnings/probeErrors mirror the WARNING/ERROR sections, which share
// an identical shape.
type probeWarnings struct {
	Warning []probeIssue `xml:"Warning"`
}

type probeErrors struct {
	Error []probeIssue `xml:"Error"`
}

type probeIssue struct {
	Type         string   `xml:"Type"`
	LocalizedMsg string   `xml:"LocalizedMsg"`
	Args         []string `xml:"Arg"`
}
