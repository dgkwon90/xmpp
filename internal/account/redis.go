package account

import "log"

func (m Management) saveOnline(key string, val interface{}) {
	if m.UseRedis {
		if redisConnErr := m.Redis.Connect(); redisConnErr != nil {
			log.Printf("[am] redis conn err: %v\n", redisConnErr)
		} else {
			// GetData
			count, existsErr := m.Redis.ExistsKey(key)
			if existsErr != nil {
				log.Printf("[am] redis exists err: %v\n", existsErr)
			}

			if count <= 0 {
				setErr := m.Redis.SetData(key, val, 0)
				if setErr != nil {
					log.Printf("[am] redis set err: %v\n", setErr)
				}
			} else {
				//m.Redis.DelData(key)
			}
		}
	}
}

func (m Management) saveOffline(key string){
	if m.UseRedis {
		if redisConnErr := m.Redis.Connect(); redisConnErr != nil {
			log.Printf("[am] redis conn err: %v\n", redisConnErr)
		} else {
			setErr := m.Redis.DelData(key)
			if setErr != nil {
				log.Printf("[am] redis del err: %v\n", setErr)
			}
		}
	}
}