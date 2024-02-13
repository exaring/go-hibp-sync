package hibpsync

import (
	"bytes"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestExport(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := NewMockstorage(ctrl)

	storageMock.EXPECT().LoadData("00000").Return(io.NopCloser(bytes.NewReader([]byte("suffix:counter11\nsuffix:counter12"))), nil)
	storageMock.EXPECT().LoadData("00001").Return(io.NopCloser(bytes.NewReader([]byte("suffix:counter2"))), nil)
	storageMock.EXPECT().LoadData("00002").Return(io.NopCloser(bytes.NewReader([]byte("suffix:counter3"))), nil)

	buf := bytes.NewBuffer([]byte{})

	if err := export(0, 3, storageMock, buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "00000suffix:counter11\n00000suffix:counter12\n00001suffix:counter2\n00002suffix:counter3" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}