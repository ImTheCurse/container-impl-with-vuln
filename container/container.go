package container

func (cont *Container) WithWorkingDirectory(workDir string) *Container {
	cont.cmd.Dir = workDir
	return cont
}

func (cont *Container) Run() error {
	err := cont.cmd.Run()
	if err != nil {
		return CmdRunFailedError
	}
	return nil
}
