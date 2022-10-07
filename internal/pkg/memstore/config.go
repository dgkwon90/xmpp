package memstore

type AddressInfo struct {
	Host string
	Port int
}

type Config struct {
	Timeout  int
	Pool     int
	Password string
	Address  AddressInfo
	Clusters string
}
