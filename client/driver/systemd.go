package driver

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/client/config"
	dstructs "github.com/hashicorp/nomad/client/driver/structs"
	"github.com/hashicorp/nomad/client/fingerprint"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/helper/fields"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/mapstructure"
)

type SystemdDriver struct {
	DriverContext
	// TODO: determine if this is the correct fingerprinter
	fingerprint.StaticFingerprinter
}

type SystemdDriverConfig struct {
	Service string `mapstructure:"service"`
}

func NewSystemdDriver(ctx *DriverContext) Driver {
	return &SystemdDriver{DriverContext: *ctx}
}

func (d *SystemdDriver) Prestart(*ExecContext, *structs.Task) (*PrestartResponse, error) {
	return nil, nil
}

func (d *SystemdDriver) Start(ctx *ExecContext, task *structs.Task) (DriverHandle, error) {
	var driverConfig SystemdDriverConfig
	if err := mapstructure.WeakDecode(task.Config, &driverConfig); err != nil {
		return nil, err
	}

	if err := exec.Command("service", driverConfig.Service, "start").Run(); err != nil {
		return nil, err
	}

	return &systemdHandle{
		serviceName: driverConfig.Service,
	}, nil
}

func (d *SystemdDriver) Open(ctx *ExecContext, handleID string) (DriverHandle, error) {
	return nil, nil
}

func (d *SystemdDriver) Cleanup(*ExecContext, *CreatedResources) error {
	return nil
}

func (d *SystemdDriver) Validate(config map[string]interface{}) error {
	fd := &fields.FieldData{
		Raw: config,
		Schema: map[string]*fields.FieldSchema{
			"service": &fields.FieldSchema{
				Type:     fields.TypeString,
				Required: true,
			},
		},
	}

	return fd.Validate()
}

func (d *SystemdDriver) Abilities() DriverAbilities {
	return DriverAbilities{
		SendSignals: false,
		Exec:        false,
	}
}

func (d *SystemdDriver) FSIsolation() cstructs.FSIsolation {
	return cstructs.FSIsolationNone
}

func (d *SystemdDriver) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	output, err := exec.Command("cat", "/proc/1/comm").Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == "systemd", nil
}

type systemdHandle struct {
	serviceName string
}

func (h *systemdHandle) ID() string {
	return h.serviceName
}

func (h *systemdHandle) WaitCh() chan *dstructs.WaitResult {
	// A wait channel doesn't really make sense for systemd services.
	// It could be implemented by periodically checking the status
	// of the service, maybe.
	return make(chan *dstructs.WaitResult)
}

func (h *systemdHandle) Update(task *structs.Task) error {
	return nil
}

func (h *systemdHandle) Kill() error {
	return exec.Command("service", h.serviceName, "stop").Run()
}

func (h *systemdHandle) Stats() (*cstructs.TaskResourceUsage, error) {
	return nil, nil
}

func (h *systemdHandle) Signal(s os.Signal) error {
	output, err := exec.Command("systemctl", "show", "-p", "MainPID", h.serviceName).Output()
	if err != nil {
		return err
	}

	parts := strings.Split(string(output), "=")
	pid, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return proc.Signal(s)
}

func (h *systemdHandle) Exec(ctx context.Context, cmd string, args []string) ([]byte, int, error) {
	return nil, 0, nil
}
