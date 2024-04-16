package service

func (s *Service) DeleteSetting(key string) error {
	return s.r.DeleteSetting(key)
}

func (s *Service) GetSettingInt32(key string) (int32, error) {
	return s.r.GetSettingInt32(key)
}

func (s *Service) SetSettingInt32(key string, value int32) error {
	return s.r.SetSettingInt32(key, value)
}
