package dao

func RedisUsrTokenKey(name string) string {
	return "usr_token:" + name
}
