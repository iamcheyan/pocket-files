package main

// StorageKind names the concrete provider family advertised by the registry.
type StorageKind string

const (
	KindInternal StorageKind = "internal"
	KindTF       StorageKind = "tf"
	KindUSB      StorageKind = "usb"
	KindSMB      StorageKind = "smb"
	KindWebDAV   StorageKind = "webdav"
	KindCloud    StorageKind = "cloud"
)

// InternalStorageProvider serves the built-in volume.
type InternalStorageProvider struct{ *LocalProvider }

// TFCardProvider serves a removable memory card.
type TFCardProvider struct{ *LocalProvider }

// USBStorageProvider serves a USB mass-storage volume.
type USBStorageProvider struct{ *LocalProvider }

func NewInternalStorageProvider(id, name, root string, readOnly bool) *InternalStorageProvider {
	return &InternalStorageProvider{NewLocalProvider(id, name, string(KindInternal), root, readOnly)}
}
func NewTFCardProvider(id, name, root string, readOnly bool) *TFCardProvider {
	return &TFCardProvider{NewLocalProvider(id, name, string(KindTF), root, readOnly)}
}
func NewUSBStorageProvider(id, name, root string, readOnly bool) *USBStorageProvider {
	return &USBStorageProvider{NewLocalProvider(id, name, string(KindUSB), root, readOnly)}
}
