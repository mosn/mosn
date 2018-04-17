package registry

import (
    "github.com/nu7hatch/gouuid"
    "time"
)

func RandomUuid() string {
    u, _ := uuid.NewV4()
    return u.String()
}

func CalRetreatTime(t int64, maxTime int64) time.Duration {
    if t == 0 {
        return time.Duration(0)
    }
    if t > maxTime {
        //5min
        return time.Duration(5 * 60 * 1000 * 1000 * 1000)
    }
    r := 1 << uint(t)
    return time.Duration(r * 1000 * 1000 * 1000)
}
