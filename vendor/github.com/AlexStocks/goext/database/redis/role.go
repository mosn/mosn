// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2017-08-12 11:57
// Package gxredis provides a redis driver by sentinel
// ref: https://github.com/alexstocks/go-sentinel/blob/master/role.go
package gxredis

import (
	"github.com/garyburd/redigo/redis"
	"github.com/juju/errors"
)

// CheckRole wraps GetRole in a test to verify if the role matches an expected
// role string. If there was any error in querying the supplied connection,
// the function returns false. Works with Redis >= 2.8.12.
// It's not goroutine safe, but if you call this method on pooled connections
// then you are OK.
func CheckRole(c redis.Conn, redisRole RedisRole) bool {
	role, err := getRole(c)
	if err != nil || getRedisRole(role) != redisRole {
		return false
	}
	return true
}

// getRole is a convenience function supplied to query an instance (master or
// slave or sentinel) for its role. It attempts to use the ROLE command introduced in
// redis 2.8.12.
func getRole(c redis.Conn) (string, error) {
	res, err := c.Do("ROLE")
	if err != nil {
		return "", err
	}
	rres, ok := res.([]interface{})
	if ok {
		return redis.String(rres[0], nil)
	}
	return "", errors.New("redigo: can not transform ROLE reply to string")
}

// getRoles is a convenience function supplied to query an instance
// for its role. It attempts to use the ROLE command introduced in
// redis 2.8.12.
func getSentileRoles(c redis.Conn) ([]string, error) {
	res, err := c.Do("ROLE")
	if err != nil {
		return []string{""}, err
	}
	res1, ok := res.([]interface{})
	if !ok {
		return []string{""}, errors.New("redigo: can not transform ROLE reply to string")
	}
	if len(res1) != 2 {
		return []string{""}, errors.New("redigo: the length of ROLE reply is not 2")
	}
	if getRedisRole(string(res1[0].([]byte))) != (RedisRole(RR_Sentinel)) {
		return []string{""}, errors.New("redigo: res[0] of ROLE replay is not \"sentinel\"")
	}
	res2, ok := res1[1].([]interface{})
	if !ok {
		return []string{""}, errors.New("redigo: can not transform res[1] of ROLE replay to []interface{}")
	}
	if len(res2) == 0 {
		return []string{""}, errors.New("redigo: the length of res[1] of ROLE reply is 0")
	}

	var addrs []string
	for _, e := range res2 {
		addrs = append(addrs, string(e.([]byte)))
	}

	return addrs, nil
}
