package geometry

type Client struct {
	pythonBin    string
	runnerScript string
}

func NewClient(pythonBin, runnerScript string) *Client {
	return &Client{
		pythonBin:    pythonBin,
		runnerScript: runnerScript,
	}
}
