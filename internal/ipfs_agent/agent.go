package ipfs_agent

type IpfsAgent interface {
	PinFile(filepath string) (string, error)
	PinContent([]byte) (string, error)
}
