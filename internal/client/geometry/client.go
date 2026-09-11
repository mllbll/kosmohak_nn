package geometry

type pythonClient struct {
	pythonBin    string
	runnerScript string
}

func NewClient(pythonBin, runnerScript string) *pythonClient {
	return &pythonClient{
		pythonBin:    pythonBin,
		runnerScript: runnerScript,
	}
}

var _ Client = (*pythonClient)(nil)
