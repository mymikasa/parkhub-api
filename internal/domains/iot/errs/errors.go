package errs

import "errors"

var (
	ErrDeviceNotFound    = errors.New("设备不存在")
	ErrDeviceIDDuplicate = errors.New("设备序列号已存在")
	ErrDeviceNotBound    = errors.New("设备当前未绑定")
	ErrDeviceMustUnbind  = errors.New("设备已绑定，请先解绑后删除")
	ErrDeviceOffline     = errors.New("设备离线请检查心跳")
	ErrInvalidCommand    = errors.New("无效的控制指令")
)
