package container

// WithWorkingDirectory sets the command working directory and returns the container.
func (cont *Container) WithWorkingDirectory(workDir string) *Container {
	cont.cmd.Dir = workDir
	return cont
}

// Run executes the container command and wraps run errors.
func (cont *Container) Run() error {
	err := cont.cmd.Run()
	if err != nil {
		return CmdRunFailedError
	}
	return nil
}
