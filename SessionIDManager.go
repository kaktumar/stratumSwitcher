package main

import (
	"sync"

	"github.com/willf/bitset"
)

//////////////////////////////// SessionIDManager //////////////////////////////

// SessionIDMask 会话ID掩码，用于分离serverID和sessionID
// 也是sessionID部分可以达到的最大数值
const SessionIDMask uint32 = 0x00FFFFFF // 16777215

// MaxValidSessionID 最大的合法sessionID
const MaxValidSessionID uint32 = SessionIDMask - 1 // 16777214

// SessionIDManager 线程安全的会话ID管理器
type SessionIDManager struct {
	//
	//  SESSION ID: UINT32
	//
	//   xxxxxxxx     xxxxxxxx xxxxxxxx xxxxxxxx
	//  ----------    --------------------------
	//  server ID          session id
	//   [1, 255]        range: [0, MaxSessionID]
	//
	serverID   uint8
	sessionIDs *bitset.BitSet

	count    uint32 // how many ids are used now
	allocIDx uint32
	lock     sync.Mutex
}

// NewSessionIDManager 创建一个会话ID管理器实例
func NewSessionIDManager(serverID uint8) *SessionIDManager {
	manager := new(SessionIDManager)

	manager.serverID = serverID
	manager.sessionIDs = bitset.New(uint(SessionIDMask))
	manager.count = 0
	manager.allocIDx = 0

	manager.sessionIDs.ClearAll()

	return manager
}

// isFull 判断会话ID是否已满（内部使用，不加锁）
func (manager *SessionIDManager) isFull() bool {
	return (manager.count >= SessionIDMask)
}

// IsFull 判断会话ID是否已满
func (manager *SessionIDManager) IsFull() bool {
	defer manager.lock.Unlock()
	manager.lock.Lock()

	return manager.isFull()
}

// AllocSessionID 为调用者分配一个会话ID
func (manager *SessionIDManager) AllocSessionID() (uint32, bool) {
	defer manager.lock.Unlock()
	manager.lock.Lock()

	if manager.isFull() {
		return SessionIDMask, false
	}

	// find an empty bit
	for manager.sessionIDs.Test(uint(manager.allocIDx)) {
		manager.allocIDx++
		if manager.allocIDx > MaxValidSessionID {
			manager.allocIDx = 0
		}
	}

	// set to true
	manager.sessionIDs.Set(uint(manager.allocIDx))
	manager.count++

	sessionID := (uint32(manager.serverID) << 24) | manager.allocIDx
	return sessionID, true
}

// FreeSessionID 释放调用者持有的会话ID
func (manager *SessionIDManager) FreeSessionID(sessionID uint32) {
	defer manager.lock.Unlock()
	manager.lock.Lock()

	idx := sessionID & SessionIDMask
	manager.sessionIDs.Clear(uint(idx))
	manager.count--
}