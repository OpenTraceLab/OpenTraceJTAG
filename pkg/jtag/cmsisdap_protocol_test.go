package jtag

import (
	"bytes"
	"testing"
)

func TestProtocolEncodeInfo(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name   string
		infoID byte
		want   []byte
	}{
		{"Vendor ID", InfoVendorID, []byte{0x00, 0x01}},
		{"Product ID", InfoProductID, []byte{0x00, 0x02}},
		{"Serial Number", InfoSerialNum, []byte{0x00, 0x03}},
		{"Firmware Version", InfoFirmwareVer, []byte{0x00, 0x04}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proto.EncodeInfo(tt.infoID)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtocolDecodeInfo(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name    string
		resp    []byte
		want    string
		wantErr bool
	}{
		{
			name: "valid vendor",
			resp: []byte{0x00, 0x04, 'T', 'e', 's', 't'},
			want: "Test",
		},
		{
			name:    "too short",
			resp:    []byte{0x00},
			wantErr: true,
		},
		{
			name:    "wrong command",
			resp:    []byte{0x01, 0x04, 'T', 'e', 's', 't'},
			wantErr: true,
		},
		{
			name:    "incomplete string",
			resp:    []byte{0x00, 0x10, 'T', 'e', 's', 't'},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := proto.DecodeInfo(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DecodeInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtocolEncodeConnect(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name string
		port byte
		want []byte
	}{
		{"Default", PortDefault, []byte{0x02, 0x00}},
		{"SWD", PortSWD, []byte{0x02, 0x01}},
		{"JTAG", PortJTAG, []byte{0x02, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proto.EncodeConnect(tt.port)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeConnect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtocolDecodeConnect(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name    string
		resp    []byte
		want    byte
		wantErr bool
	}{
		{
			name: "JTAG connected",
			resp: []byte{0x02, 0x02},
			want: PortJTAG,
		},
		{
			name: "SWD connected",
			resp: []byte{0x02, 0x01},
			want: PortSWD,
		},
		{
			name:    "connection failed",
			resp:    []byte{0x02, 0x00},
			wantErr: true,
		},
		{
			name:    "too short",
			resp:    []byte{0x02},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := proto.DecodeConnect(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeConnect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DecodeConnect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtocolEncodeJTAGConfigure(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name      string
		irLengths []byte
		want      []byte
	}{
		{
			name:      "single device",
			irLengths: []byte{5},
			want:      []byte{0x15, 0x01, 0x05},
		},
		{
			name:      "two devices",
			irLengths: []byte{5, 8},
			want:      []byte{0x15, 0x02, 0x05, 0x08},
		},
		{
			name:      "three devices",
			irLengths: []byte{5, 8, 4},
			want:      []byte{0x15, 0x03, 0x05, 0x08, 0x04},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proto.EncodeJTAGConfigure(tt.irLengths)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeJTAGConfigure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewJTAGSequence(t *testing.T) {
	tests := []struct {
		name       string
		tckCount   int
		tms        bool
		captureTDO bool
		tdi        []byte
		wantInfo   byte
	}{
		{
			name:       "8 clocks, TMS=0, no capture",
			tckCount:   8,
			tms:        false,
			captureTDO: false,
			tdi:        []byte{0xAA},
			wantInfo:   0x08,
		},
		{
			name:       "8 clocks, TMS=1, capture TDO",
			tckCount:   8,
			tms:        true,
			captureTDO: true,
			tdi:        []byte{0xAA},
			wantInfo:   0x08 | 0x40 | 0x80,
		},
		{
			name:       "64 clocks (0 means 64)",
			tckCount:   64,
			tms:        false,
			captureTDO: false,
			tdi:        []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantInfo:   0x00, // 64 is encoded as 0
		},
		{
			name:       "5 clocks, TMS=1, no capture",
			tckCount:   5,
			tms:        true,
			captureTDO: false,
			tdi:        []byte{0x1F},
			wantInfo:   0x05 | 0x40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := NewJTAGSequence(tt.tckCount, tt.tms, tt.captureTDO, tt.tdi)

			if seq.Info != tt.wantInfo {
				t.Errorf("Info = 0x%02X, want 0x%02X", seq.Info, tt.wantInfo)
			}

			// Test getter methods
			if seq.TMS() != tt.tms {
				t.Errorf("TMS() = %v, want %v", seq.TMS(), tt.tms)
			}

			if seq.CaptureTDO() != tt.captureTDO {
				t.Errorf("CaptureTDO() = %v, want %v", seq.CaptureTDO(), tt.captureTDO)
			}

			expectedCount := tt.tckCount
			if tt.tckCount == 64 {
				expectedCount = 64
			}
			if seq.TCKCount() != expectedCount {
				t.Errorf("TCKCount() = %v, want %v", seq.TCKCount(), expectedCount)
			}
		})
	}
}

func TestProtocolEncodeJTAGSequence(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name      string
		sequences []JTAGSequence
		want      []byte
	}{
		{
			name: "single sequence",
			sequences: []JTAGSequence{
				NewJTAGSequence(8, false, true, []byte{0xAA}),
			},
			want: []byte{0x14, 0x01, 0x88, 0xAA},
		},
		{
			name: "two sequences",
			sequences: []JTAGSequence{
				NewJTAGSequence(8, false, true, []byte{0xAA}),
				NewJTAGSequence(5, true, false, []byte{0x1F}),
			},
			want: []byte{0x14, 0x02, 0x88, 0xAA, 0x45, 0x1F},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proto.EncodeJTAGSequence(tt.sequences)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeJTAGSequence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtocolDecodeJTAGSequence(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name      string
		resp      []byte
		sequences []JTAGSequence
		want      [][]byte
		wantErr   bool
	}{
		{
			name: "single sequence with TDO",
			resp: []byte{0x14, 0x00, 0xFF},
			sequences: []JTAGSequence{
				NewJTAGSequence(8, false, true, []byte{0xAA}),
			},
			want: [][]byte{{0xFF}},
		},
		{
			name: "two sequences, one with TDO",
			resp: []byte{0x14, 0x00, 0xFF},
			sequences: []JTAGSequence{
				NewJTAGSequence(8, false, true, []byte{0xAA}),
				NewJTAGSequence(5, true, false, []byte{0x1F}),
			},
			want: [][]byte{{0xFF}},
		},
		{
			name:    "error status",
			resp:    []byte{0x14, 0xFF},
			sequences: []JTAGSequence{
				NewJTAGSequence(8, false, true, []byte{0xAA}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := proto.DecodeJTAGSequence(tt.resp, tt.sequences)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJTAGSequence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("DecodeJTAGSequence() returned %d sequences, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if !bytes.Equal(got[i], tt.want[i]) {
					t.Errorf("Sequence %d: got %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestProtocolEncodeSetClock(t *testing.T) {
	proto := NewCMSISDAPProtocol(64)

	tests := []struct {
		name string
		hz   uint32
		want []byte
	}{
		{
			name: "1 MHz",
			hz:   1_000_000,
			want: []byte{0x11, 0x40, 0x42, 0x0F, 0x00},
		},
		{
			name: "10 MHz",
			hz:   10_000_000,
			want: []byte{0x11, 0x80, 0x96, 0x98, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proto.EncodeSetClock(tt.hz)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeSetClock() = %v, want %v", got, tt.want)
			}
		})
	}
}
