package service

import "context"

func (s *Service) DeleteSetting(key string) error {
	ctx := context.Background()
	return s.r.DeleteSetting(ctx, key)
}

func (s *Service) GetSettingInt32(key string) (int32, error) {
	ctx := context.Background()
	return s.r.GetSettingInt32(ctx, key)
}

func (s *Service) GetSettingInt(key string) (int, error) {
	ctx := context.Background()
	return s.r.GetSettingInt(ctx, key)
}

func (s *Service) SetSettingInt32(key string, value int32) error {
	ctx := context.Background()
	return s.r.SetSettingInt32(ctx, key, value)
}

func (s *Service) SetSettingInt(key string, value int) error {
	ctx := context.Background()
	return s.r.SetSettingInt(ctx, key, value)
}
