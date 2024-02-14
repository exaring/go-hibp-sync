package hibpsync

import (
	"github.com/alitto/pond"
	"github.com/h2non/gock"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"go.uber.org/mock/gomock"
)

const baseURL = "https://api.pwnedpasswords.com"

func TestSync(t *testing.T) {
	httpClient := &http.Client{}
	gock.InterceptClient(httpClient)

	gock.New(baseURL).
		Get("/range/00000").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix1")
	gock.New(baseURL).
		Get("/range/00001").
		MatchHeader("If-None-Match", "etag received earlier").
		Reply(http.StatusNotModified).
		AddHeader("ETag", "etag received earlier").
		BodyString("suffix2")
	gock.New(baseURL).
		Get("/range/00002").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix31:2\nsuffix32:3")
	gock.New(baseURL).
		Get("/range/00003").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix4")
	gock.New(baseURL).
		Get("/range/00004").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix5")
	gock.New(baseURL).
		Get("/range/00005").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix6")
	gock.New(baseURL).
		Get("/range/00006").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix7")
	gock.New(baseURL).
		Get("/range/00007").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix8")
	gock.New(baseURL).
		Get("/range/00008").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix9")
	gock.New(baseURL).
		Get("/range/00009").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix10")
	gock.New(baseURL).
		Get("/range/0000A").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix11")
	gock.New(baseURL).
		Get("/range/0000B").
		Reply(200).
		AddHeader("ETag", "etag").
		BodyString("suffix12")

	client := &hibpClient{
		endpoint:   defaultEndpoint,
		httpClient: httpClient,
	}

	ctrl := gomock.NewController(t)
	storageMock := NewMockstorage(ctrl)

	storageMock.EXPECT().LoadETag("00000").Return("", nil)
	storageMock.EXPECT().Save("00000", "etag", []byte("00000suffix1")).Return(nil)
	storageMock.EXPECT().LoadETag("00001").Return("etag received earlier", nil)
	// 00001 does not need to be written as its ETag has not changed
	storageMock.EXPECT().LoadETag("00002").Return("", nil)
	storageMock.EXPECT().Save("00002", "etag", []byte("00002suffix31:2\n00002suffix32:3")).Return(nil)
	storageMock.EXPECT().LoadETag("00003").Return("", nil)
	storageMock.EXPECT().Save("00003", "etag", []byte("00003suffix4")).Return(nil)
	storageMock.EXPECT().LoadETag("00004").Return("", nil)
	storageMock.EXPECT().Save("00004", "etag", []byte("00004suffix5")).Return(nil)
	storageMock.EXPECT().LoadETag("00005").Return("", nil)
	storageMock.EXPECT().Save("00005", "etag", []byte("00005suffix6")).Return(nil)
	storageMock.EXPECT().LoadETag("00006").Return("", nil)
	storageMock.EXPECT().Save("00006", "etag", []byte("00006suffix7")).Return(nil)
	storageMock.EXPECT().LoadETag("00007").Return("", nil)
	storageMock.EXPECT().Save("00007", "etag", []byte("00007suffix8")).Return(nil)
	storageMock.EXPECT().LoadETag("00008").Return("", nil)
	storageMock.EXPECT().Save("00008", "etag", []byte("00008suffix9")).Return(nil)
	storageMock.EXPECT().LoadETag("00009").Return("", nil)
	storageMock.EXPECT().Save("00009", "etag", []byte("00009suffix10")).Return(nil)
	storageMock.EXPECT().LoadETag("0000A").Return("", nil)
	storageMock.EXPECT().Save("0000A", "etag", []byte("0000Asuffix11")).Return(nil)
	storageMock.EXPECT().LoadETag("0000B").Return("", nil)
	storageMock.EXPECT().Save("0000B", "etag", []byte("0000Bsuffix12")).Return(nil)

	var callCounter atomic.Int64

	progressFn := func(_, _, _, processed, _ int64) error {
		callCounter.Add(1)

		if processed != 10 && processed != 12 {
			t.Fatalf("progressFn called at unexpected point in time: %d", processed)
		}

		return nil
	}

	// Create the pool with some arbitrary configuration
	pool := pond.New(3, 3)

	if err := sync(0, 12, client, storageMock, pool, progressFn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCounter.Load() != 2 {
		t.Fatalf("unexpected number of calls to progressFn: %d", callCounter.Load())
	}

	if !ctrl.Satisfied() {
		t.Fatalf("there are unsatisfied expectations")
	}

	if gock.IsPending() {
		t.Fatalf("there are pending mocks")
	}
}

// TODO: We will need further testcases ensuring the library works fine even in error conditions

// Code generated by MockGen. DO NOT EDIT.
// Source: storage.go
//
// Generated by this command:
//
//	mockgen -source storage.go
//

// Mockstorage is a mock of storage interface.
type Mockstorage struct {
	ctrl     *gomock.Controller
	recorder *MockstorageMockRecorder
}

// MockstorageMockRecorder is the mock recorder for Mockstorage.
type MockstorageMockRecorder struct {
	mock *Mockstorage
}

// NewMockstorage creates a new mock instance.
func NewMockstorage(ctrl *gomock.Controller) *Mockstorage {
	mock := &Mockstorage{ctrl: ctrl}
	mock.recorder = &MockstorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockstorage) EXPECT() *MockstorageMockRecorder {
	return m.recorder
}

// LoadData mocks base method.
func (m *Mockstorage) LoadData(key string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadData", key)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadData indicates an expected call of LoadData.
func (mr *MockstorageMockRecorder) LoadData(key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadData", reflect.TypeOf((*Mockstorage)(nil).LoadData), key)
}

// LoadETag mocks base method.
func (m *Mockstorage) LoadETag(key string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadETag", key)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadETag indicates an expected call of LoadETag.
func (mr *MockstorageMockRecorder) LoadETag(key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadETag", reflect.TypeOf((*Mockstorage)(nil).LoadETag), key)
}

// Save mocks base method.
func (m *Mockstorage) Save(key, etag string, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", key, etag, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockstorageMockRecorder) Save(key, etag, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*Mockstorage)(nil).Save), key, etag, data)
}
