package deploywizard

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/deploy"
)

// newTextInput builds a plain (or password-masked) textinput.Model
// pre-filled with value, used across the settings/advanced steps.
func newTextInput(value string, password bool) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	ti.CharLimit = 256
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	return ti
}

func morefRef(ref types.ManagedObjectReference) config.MorefRef {
	return config.MorefRef{Type: ref.Type, Value: ref.Value}
}

// profileFromSpec captures a spec's reusable target settings (everything
// except connection/source specifics) as a named DeploymentProfile.
func profileFromSpec(name string, spec deploy.Spec) config.DeploymentProfile {
	p := config.DeploymentProfile{
		Name:                name,
		ComputeResourceName: spec.ComputeResourceName,
		ResourcePoolName:    spec.ResourcePoolName,
		DiskMode:            string(spec.DiskMode),
		PowerOn:             spec.PowerOn,
		ImportAsTemplate:    spec.ImportAsTemplate,
		Properties:          spec.Properties,
		ExtraArgs:           spec.ExtraArgs,
	}
	if spec.DatastoreRef.Value != "" {
		ref := morefRef(spec.DatastoreRef)
		p.Datastore = &ref
	}
	if spec.FolderRef != nil {
		ref := morefRef(*spec.FolderRef)
		p.Folder = &ref
	}
	if len(spec.Networks) > 0 {
		p.NetworkMap = make(map[string]config.MorefRef, len(spec.Networks))
		for _, n := range spec.Networks {
			p.NetworkMap[n.OVFName] = morefRef(n.TargetRef)
		}
	}
	return p
}
