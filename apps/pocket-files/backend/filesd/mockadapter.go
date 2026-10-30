package main

// allDevices satisfies storageBackend for the mock provider so the backend and
// tests can run against mock storage alone.
func (p *MockProvider) allDevices() []Device {
	return append([]Device(nil), p.devices...)
}

// resolve satisfies storageBackend using the same sandbox rules as MockProvider
// operations.
func (p *MockProvider) resolve(storageID, rel string, allowMissing bool) (string, error) {
	return (sandbox{p.roots}).resolve(storageID, rel, allowMissing)
}
