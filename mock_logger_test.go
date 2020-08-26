// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/asecurityteam/runhttp (interfaces: Logger)

// Package transport is a generated GoMock package.
package transport

import (
	reflect "reflect"

	logevent "github.com/asecurityteam/logevent"
	gomock "github.com/golang/mock/gomock"
)

type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
}

// MockLoggerMockRecorder is the mock recorder for MockLogger
type MockLoggerMockRecorder struct {
	mock *MockLogger
}

// NewMockLogger creates a new mock instance
func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

// Copy mocks base method
func (m *MockLogger) Copy() logevent.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Copy")
	ret0, _ := ret[0].(logevent.Logger)
	return ret0
}

// Copy indicates an expected call of Copy
func (mr *MockLoggerMockRecorder) Copy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Copy", reflect.TypeOf((*MockLogger)(nil).Copy))
}

// Debug mocks base method
func (m *MockLogger) Debug(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Debug", arg0)
}

// Debug indicates an expected call of Debug
func (mr *MockLoggerMockRecorder) Debug(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debug", reflect.TypeOf((*MockLogger)(nil).Debug), arg0)
}

// Error mocks base method
func (m *MockLogger) Error(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Error", arg0)
}

// Error indicates an expected call of Error
func (mr *MockLoggerMockRecorder) Error(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockLogger)(nil).Error), arg0)
}

// Info mocks base method
func (m *MockLogger) Info(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Info", arg0)
}

// Info indicates an expected call of Info
func (mr *MockLoggerMockRecorder) Info(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockLogger)(nil).Info), arg0)
}

// SetField mocks base method
func (m *MockLogger) SetField(arg0 string, arg1 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetField", arg0, arg1)
}

// SetField indicates an expected call of SetField
func (mr *MockLoggerMockRecorder) SetField(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetField", reflect.TypeOf((*MockLogger)(nil).SetField), arg0, arg1)
}

// Warn mocks base method
func (m *MockLogger) Warn(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Warn", arg0)
}

// Warn indicates an expected call of Warn
func (mr *MockLoggerMockRecorder) Warn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warn", reflect.TypeOf((*MockLogger)(nil).Warn), arg0)
}
