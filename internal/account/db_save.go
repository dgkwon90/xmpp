package account

import "log"

func (m Management) saveOnline(key string, val string) {
	if m.UseDB {
		if redisConnErr := m.DB.Connect(); redisConnErr != nil {
			log.Printf("[am][db] redis conn err: %v\n", redisConnErr)
		} else {
			// GetData
			count, existsErr := m.DB.ExistsKey(key)
			if existsErr != nil {
				log.Printf("[am][db] redis exists err: %v\n", existsErr)
			}

			if count <= 0 {
				setErr := m.DB.SetData(key, val, 0)
				if setErr != nil {
					log.Printf("[am][db] redis set err: %v\n", setErr)
				}
			} else {
				//m.DB.DelData(key)
			}
		}
	}
}

func (m Management) saveOffline(key string) {
	if m.UseDB {
		if redisConnErr := m.DB.Connect(); redisConnErr != nil {
			log.Printf("[am][db] redis conn err: %v\n", redisConnErr)
		} else {
			setErr := m.DB.DelData(key)
			if setErr != nil {
				log.Printf("[am][db] redis del err: %v\n", setErr)
			}
		}
	}
}
