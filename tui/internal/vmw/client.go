package vmw

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// VMClient defines operations the TUI can perform against the vmw CLI.
// Identifier arguments accept a machine ID, vagrantfile path, or VM name.
type VMClient interface {
	Status() (StatusResponse, error)
	Extend(identifier, duration string) error
	Halt(identifier string) error
	Destroy(identifier string) error
	Exempt(identifier string) error
	Sweep() error
	Update(identifier string, provision bool) error
	UpdateAll(provision bool) error
	Peek(identifier string) (string, error)
}

// ExecClient implements VMClient by shelling out to the vmw binary.
type ExecClient struct {
	VmwPath string
}

func (c *ExecClient) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.VmwPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w\n%s", c.VmwPath, args, err, out)
	}
	return out, nil
}

func (c *ExecClient) Status() (StatusResponse, error) {
	out, err := c.run("status", "--json")
	if err != nil {
		return StatusResponse{}, err
	}
	var resp StatusResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return StatusResponse{}, fmt.Errorf("parse status JSON: %w", err)
	}
	return resp, nil
}

func (c *ExecClient) Extend(identifier, duration string) error {
	_, err := c.run("extend", identifier, duration)
	return err
}

func (c *ExecClient) Halt(identifier string) error {
	_, err := c.run("halt", identifier)
	return err
}

func (c *ExecClient) Destroy(identifier string) error {
	_, err := c.run("destroy", identifier)
	return err
}

func (c *ExecClient) Exempt(identifier string) error {
	_, err := c.run("exempt", identifier)
	return err
}

func (c *ExecClient) Sweep() error {
	_, err := c.run("sweep")
	return err
}

func (c *ExecClient) Update(identifier string, provision bool) error {
	args := []string{"update", identifier}
	if provision {
		args = append(args, "--provision")
	}
	_, err := c.run(args...)
	return err
}

func (c *ExecClient) Peek(identifier string) (string, error) {
	out, err := c.run("peek", identifier)
	return string(out), err
}

func (c *ExecClient) UpdateAll(provision bool) error {
	args := []string{"update", "--all"}
	if provision {
		args = append(args, "--provision")
	}
	_, err := c.run(args...)
	return err
}
