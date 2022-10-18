package account


func (m Management) notifyOnline(endpoint string) {
	if m.UseMQ {
		m.Publisher.Push()
	}
}

func (m Management) notifyOffline(endpoint string) {
	if m.UseMQ {
		m.Publisher.Push()
	}
}