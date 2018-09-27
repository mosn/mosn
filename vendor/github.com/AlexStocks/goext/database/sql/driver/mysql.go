// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2016-09-11 19:30
// Package gxdriver provides a MySQL driver for Go's database/sql package
// code example: https://github.com/alexstocks/go-practice/blob/master/mysql/stmt.go
package gxdriver

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	// "github.com/AlexStocks/goext/log"
	"github.com/AlexStocks/goext/strings"
	"github.com/go-sql-driver/mysql"
)

const (
	GettyMySQLDriver = "getty_mysql_driver"
)

func init() {
	sql.Register(GettyMySQLDriver, &MySQLDriver{})
	dbConnMap = make(map[string]*sql.DB)
}

///////////////////////////////////////
// Mysql Driver
///////////////////////////////////////

// refs: https://zhuanlan.zhihu.com/p/24768377
// author: https://github.com/idada
// 重写driver，方便重用prepare statement.
// !: statement用到的connection和transaction begin的connection不是同一个connection
type mysqlStmt struct {
	driver.Stmt
	conn  *mySQLConn
	query string
	ref   int32
}

// 程序退出的时候调用之
func (stmt *mysqlStmt) Close() error {
	// gxlog.CInfo("Close()")
	stmt.conn.stmtMutex.Lock()
	defer stmt.conn.stmtMutex.Unlock()

	if atomic.AddInt32(&stmt.ref, -1) == 0 {
		// gxlog.CInfo("really close")
		delete(stmt.conn.stmtCache, stmt.query)
		return stmt.Stmt.Close()
	}
	return nil
}

type MySQLDriver struct {
}

func (d MySQLDriver) Open(dsn string) (driver.Conn, error) {
	// gxlog.CInfo("GettyMSDriver:Open(%s)", dsn)
	var driver mysql.MySQLDriver
	conn, err := driver.Open(dsn)
	if err != nil {
		return nil, err
	}

	return &mySQLConn{Conn: conn, stmtCache: make(map[string]*mysqlStmt)}, nil
}

type mySQLConn struct {
	driver.Conn
	stmtMutex sync.RWMutex
	// 缓存prepared stmt链接，在程序启动的时候全都创建好
	stmtCache map[string]*mysqlStmt
}

// 这个函数应该在程序启动的时候被调用以创建全局prepared stmt句柄，在程序退出的时候调用(stmt *mysqlStmt) Close()
func (m *mySQLConn) Prepare(query string) (driver.Stmt, error) {
	// gxlog.CInfo("GettyMSDriver:Prepare(%s)", query)
	m.stmtMutex.RLock()
	if stmt, exists := m.stmtCache[query]; exists {
		// must update reference counter in lock scope
		atomic.AddInt32(&stmt.ref, 1)
		m.stmtMutex.RUnlock()
		return stmt, nil
	}
	m.stmtMutex.RUnlock()

	m.stmtMutex.Lock()
	defer m.stmtMutex.Unlock()

	// double check
	if stmt, exists := m.stmtCache[query]; exists {
		atomic.AddInt32(&stmt.ref, 1)
		return stmt, nil
	}

	stmt, err := m.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}

	stmt2 := &mysqlStmt{stmt, m, query, 1}
	m.stmtCache[query] = stmt2
	return stmt2, nil
}

func (m *mySQLConn) Begin() (driver.Tx, error) {
	// gxlog.CInfo("GettyMSDriver:Begin")
	tx, err := m.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return &mysqlTx{m, tx}, nil
}

type mysqlTx struct {
	*mySQLConn
	tx driver.Tx
}

func (tx *mysqlTx) Commit() (err error) {
	// gxlog.CInfo("GettyMSDriver:Commit")
	return tx.tx.Commit()
}

func (tx *mysqlTx) Rollback() (err error) {
	return tx.tx.Rollback()
}

///////////////////////////////////////
// Mysql Instance
///////////////////////////////////////

var (
	dbConnMapLock sync.Mutex
	dbConnMap     map[string]*sql.DB
)

/*
feature list:
1 读写分离
2 长连接有效性检测和自动重连
3 mysql prepared statement 重用
! Note: 在执行sql操作之前请执行CheckSQl & CheckAcitve
*/
type MySQL struct {
	*sql.DB             //数据库操作
	tx          *sql.Tx //带事务的数据库操作
	txIndex     int     //事务记数器，只有当txIndex=0才会触发Begin动作，只有当txIndex=1才会触发Commit动作
	role        string  //操作类型，分Master或Slave
	schema      string  //数据库连接schema
	active      int64   //上次建立连接的时间点，需要一种机制来检测客户端与mysql服务端连接的有效性
	waitTimeout int64   //mysql服务器空闲等待时长
	sync.Once
}

//创建一个默认的mysql操作实例
func Open(schema string) *MySQL {
	return newMySQLInstance(schema, "Master")
}

//创建一个默认的mysql查询实例
func OpenQuery(schema string) *MySQL {
	return newMySQLInstance(schema, "Slave")
}

func (m *MySQL) Close() {
	m.Do(func() {
		if m.tx != nil {
			m.tx.Commit()
		}
		m.tx = nil
		m.DB.Close()
		var key string = m.schema + m.role
		dbConnMapLock.Lock()
		db, ok := dbConnMap[key]
		if ok && db == m.DB {
			delete(dbConnMap, key)
		}
		dbConnMapLock.Unlock()
		m.DB = nil
	})
}

//@param string schema
//@param string role [Master, Slave]
func newMySQLInstance(schema string, role string) *MySQL {
	if !gxstrings.Contains([]string{"Master", "Slave"}, role) {
		panic("function common.NewSqlInstance's second argument must be 'Master' or 'Slave'.")
	}

	var key string = schema + role
	dbConnMapLock.Lock()
	conn, ok := dbConnMap[key]
	dbConnMapLock.Unlock()
	if !ok {
		//建立一个新连接到mysql
		connect(schema, role)
		dbConnMapLock.Lock()
		conn = dbConnMap[key]
		dbConnMapLock.Unlock()
	}
	return &MySQL{
		DB:     conn,
		schema: schema,
		role:   role,
		active: time.Now().Unix(),
	}
}

//建立数据库连接
//@param string schema 连接DB方案
//@param string role 连接类型，是分Master和Slave类型
func connect(schema string, role string) {
	var key = schema + role
	//开始连接DB
	conn, err := sql.Open(GettyMySQLDriver, schema)
	if err != nil {
		panic(err)
	}

	//将DB连接放入一个全局变量中
	dbConnMapLock.Lock()
	dbConnMap[key] = conn
	dbConnMapLock.Unlock()
}

//根据db.Query的查询结果，组装成一个关联key的数据集，数据类型[]map[string]string
func (m *MySQL) FetchRows(rows *sql.Rows) ([]map[string]string, error) {
	result := make([]map[string]string, 0)
	columns, err := rows.Columns()
	if err != nil {
		//an error occurred
		return nil, err
	}

	rawBytes := make([]sql.RawBytes, len(columns))

	//rows.Scan wants '[]interface{}' as an argument, so we must copy
	//the references into such a slice
	scanArgs := make([]interface{}, len(columns))

	for i := range rawBytes {
		scanArgs[i] = &rawBytes[i]
	}

	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		var val string
		item := make(map[string]string)
		for i, col := range rawBytes {
			if col == nil {
				val = ""
			} else {
				val = string(col)
			}
			item[columns[i]] = val
		}
		result = append(result, item)
	}
	return result, nil
}

//检查MySQL实例的连接是否还在活跃时间范围内
//! note: 这个函数的实际意义在于更新active时间，实际的sql.DB下面有一个连接池，
// 完全没必要因为超时就更换conn对象。
func (m *MySQL) CheckActive() {
	var now int64 = time.Now().Unix()
	if m.tx != nil {
		//如果存在事务会话，则不再进行连接检查
		m.active = now
		return
	}

	//从MySQL的wait_timeout变量中定位waitTimeout
	if m.waitTimeout == 0 {
		rows, err := m.Query("SHOW VARIABLES LIKE 'wait_timeout'")
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		result, err := m.FetchRows(rows)
		if err != nil {
			panic(err)
		}
		if result != nil && len(result) != 0 {
			timeout, err := strconv.Atoi(result[0]["Value"])
			if err != nil {
				panic(err)
			}
			m.waitTimeout = int64(timeout)
		}
	}

	if now-m.active > m.waitTimeout-2 {
		//此时认为数据库连接已经超时了，重新进行一次连接
		connect(m.schema, m.role)
		var key string = m.schema + m.role
		dbConnMapLock.Lock()
		m.DB = dbConnMap[key]
		dbConnMapLock.Unlock()
	}

	//设置当前时间为最新活跃点
	m.active = now
}

//保证修改、写入类的操作不在slave上执行
func (m *MySQL) CheckSQL(sql string) error {
	if m.role == "Slave" {
		sql = strings.TrimSpace(sql)
		exp := regexp.MustCompile(`^(?i:insert|update|delete|alter|truncate|drop)`)
		if exp.MatchString(sql) {
			return fmt.Errorf("insert|update|delete|alter|truncate|drop operation is not allowed on slave.")
		}
	}

	return nil
}

//开始一个事务，开始一个事务和提交、回滚事务必须一一对应
func (m *MySQL) Begin() error {
	if m.txIndex == 0 {
		var err error
		m.tx, err = m.DB.Begin()
		if err != nil {
			return err
		}
	}
	m.txIndex++

	return nil
}

//提交一个事务
func (m *MySQL) Commit() error {
	var err error
	if m.txIndex == 1 {
		err = m.tx.Commit()
		m.tx = nil
	}
	m.txIndex--

	return err
}

//事务回滚
func (m *MySQL) Rollback() error {
	err := m.tx.Rollback()
	m.txIndex = 0
	m.tx = nil

	return err
}
