package bilibili

import (
	"errors"
	localdb "github.com/Sora233/DDBOT/lsp/buntdb"
	"github.com/Sora233/DDBOT/lsp/concern"
	"github.com/tidwall/buntdb"
	"strconv"
	"time"
)

type StateManager struct {
	*concern.StateManager
	*extraKey
}

func (c *StateManager) GetGroupConcernConfig(groupCode int64, id interface{}) (concernConfig concern.IConfig) {
	return NewGroupConcernConfig(c.StateManager.GetGroupConcernConfig(groupCode, id), c)
}

func (c *StateManager) AddUserInfo(userInfo *UserInfo) error {
	if userInfo == nil {
		return errors.New("nil UserInfo")
	}
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		key := c.UserInfoKey(userInfo.Mid)
		_, _, err := tx.Set(key, userInfo.ToString(), nil)
		return err
	})
}

func (c *StateManager) GetUserInfo(mid int64) (*UserInfo, error) {
	var userInfo = &UserInfo{}
	err := c.JsonGet(c.UserInfoKey(mid), userInfo)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (c *StateManager) AddUserStat(userStat *UserStat, opt *buntdb.SetOptions) error {
	if userStat == nil {
		return errors.New("nil UserStat")
	}
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(c.UserStatKey(userStat.Mid), userStat.ToString(), opt)
		return err
	})
}

func (c *StateManager) GetUserStat(mid int64) (*UserStat, error) {
	var userStat = &UserStat{}
	err := c.JsonGet(c.UserStatKey(mid), userStat)
	if err != nil {
		return nil, err
	}
	return userStat, nil
}

func (c *StateManager) AddLiveInfo(liveInfo *LiveInfo) error {
	if liveInfo == nil {
		return errors.New("nil LiveInfo")
	}
	err := c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(c.UserInfoKey(liveInfo.Mid), liveInfo.UserInfo.ToString(), nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(c.CurrentLiveKey(liveInfo.Mid), liveInfo.ToString(), localdb.ExpireOption(time.Hour*24*7))
		return err
	})
	return err
}

func (c *StateManager) GetLiveInfo(mid int64) (*LiveInfo, error) {
	var liveInfo = &LiveInfo{}
	err := c.JsonGet(c.CurrentLiveKey(mid), liveInfo)
	if err != nil {
		return nil, err
	}
	return liveInfo, nil
}

func (c *StateManager) AddNewsInfo(newsInfo *NewsInfo) error {
	if newsInfo == nil {
		return errors.New("nil NewsInfo")
	}
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(c.UserInfoKey(newsInfo.Mid), newsInfo.UserInfo.ToString(), nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(c.CurrentNewsKey(newsInfo.Mid), newsInfo.ToString(), nil)
		return err
	})
}

func (c *StateManager) DeleteNewsInfo(mid int64) error {
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(c.CurrentNewsKey(mid))
		return err
	})
}

func (c *StateManager) DeleteLiveInfo(mid int64) error {
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(c.CurrentLiveKey(mid))
		return err
	})
}

func (c *StateManager) DeleteNewsAndLiveInfo(mid int64) error {
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(c.CurrentLiveKey(mid))
		if err != nil && err != buntdb.ErrNotFound {
			return err
		}
		_, err = tx.Delete(c.CurrentNewsKey(mid))
		return err
	})
}

func (c *StateManager) ClearByMid(mid int64) error {
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		var errs []error
		_, err := tx.Delete(c.CurrentLiveKey(mid))
		errs = append(errs, err)
		_, err = tx.Delete(c.CurrentNewsKey(mid))
		errs = append(errs, err)
		_, err = tx.Delete(c.UidFirstTimestamp(mid))
		errs = append(errs, err)
		_, err = tx.Delete(c.UserInfoKey(mid))
		errs = append(errs, err)
		_, err = tx.Delete(c.NotLiveKey(mid))
		errs = append(errs, err)
		for _, e := range errs {
			if e != nil && e != buntdb.ErrNotFound {
				return e
			}
		}
		return nil
	})
}

func (c *StateManager) GetNewsInfo(mid int64) (*NewsInfo, error) {
	var newsInfo = &NewsInfo{}
	err := c.JsonGet(c.CurrentNewsKey(mid), newsInfo)
	if err != nil {
		return nil, err
	}
	return newsInfo, nil
}

func (c *StateManager) CheckDynamicId(dynamic int64) (result bool) {
	err := c.RCoverTx(func(tx *buntdb.Tx) error {
		key := c.DynamicIdKey(dynamic)
		_, err := tx.Get(key)
		if err == nil {
			result = false
		} else if err == buntdb.ErrNotFound {
			result = true
		} else {
			return err
		}
		return nil
	})
	if err != nil {
		result = false
	}
	return result
}

func (c *StateManager) MarkDynamicId(dynamic int64) (replaced bool, err error) {
	//	一个错误的写法，用闭包返回值简单地替代了RWTxCover返回值
	//	在磁盘空间用尽的情况下，闭包可以成功执行，但RWTxCover执行持久化时会报错，这个错误就被意外地忽略了
	//	c.RWCoverTx(func(tx *buntdb.Tx) error {
	//		key := c.DynamicIdKey(dynamic)
	//		_, replaced, err = tx.Set(key, "", localdb.ExpireOption(time.Hour*120))
	//		return err
	//	})
	err = c.RWCoverTx(func(tx *buntdb.Tx) error {
		var err error
		key := c.DynamicIdKey(dynamic)
		_, replaced, err = tx.Set(key, "", localdb.ExpireOption(time.Hour*120))
		return err
	})
	return
}

func (c *StateManager) IncNotLiveCount(uid int64) int64 {
	result, err := c.SeqNext(c.NotLiveKey(uid))
	if err != nil {
		result = 0
	}
	return result
}

func (c *StateManager) ClearNotLiveCount(uid int64) error {
	return c.SeqClear(c.NotLiveKey(uid))
}

func (c *StateManager) SetUidFirstTimestampIfNotExist(uid int64, timestamp int64) error {
	return c.SetIfNotExist(c.UidFirstTimestamp(uid), strconv.FormatInt(timestamp, 10), nil)
}

func (c *StateManager) UnsetUidFirstTimestamp(uid int64) error {
	return c.RWCoverTx(func(tx *buntdb.Tx) error {
		key := c.UidFirstTimestamp(uid)
		_, err := tx.Delete(key)
		return err
	})
}

func (c *StateManager) GetUidFirstTimestamp(uid int64) (timestamp int64, err error) {
	err = c.RCoverTx(func(tx *buntdb.Tx) error {
		var err error
		key := c.UidFirstTimestamp(uid)
		var tsStr string
		tsStr, err = tx.Get(key)
		if err != nil {
			return err
		}
		timestamp, err = strconv.ParseInt(tsStr, 10, 64)
		return err
	})
	if err != nil {
		timestamp = 0
	}
	return
}

func (c *StateManager) SetGroupVideoOriginMarkIfNotExist(groupCode int64, bvid string) error {
	return localdb.SetIfNotExist(
		localdb.BilibiliVideoOriginMarkKey(groupCode, bvid),
		"",
		localdb.ExpireOption(time.Minute*15),
	)
}

func (c *StateManager) SetGroupOriginMarkIfNotExist(groupCode int64, dynamicIdStr string) error {
	return localdb.SetIfNotExist(
		localdb.BilibiliOriginMarkKey(groupCode, dynamicIdStr),
		"",
		localdb.ExpireOption(time.Minute*15),
	)
}

func SetCookieInfo(username string, cookieInfo *LoginResponse_Data_CookieInfo) error {
	if cookieInfo == nil {
		return errors.New("<nil> cookieInfo")
	}
	return localdb.RWCoverTx(func(tx *buntdb.Tx) error {
		key := localdb.BilibiliUserCookieInfoKey(username)
		cookieData, err := json.Marshal(cookieInfo)
		if err != nil {
			return err
		}
		var expire int64
		for _, cookie := range cookieInfo.GetCookies() {
			expire = cookie.GetExpires()
			break
		}
		if expire != 0 {
			_, _, err = tx.Set(key, string(cookieData), localdb.ExpireOption(time.Duration(expire-time.Now().Unix())*time.Second))
		} else {
			_, _, err = tx.Set(key, string(cookieData), nil)
		}
		return err
	})
}

func GetCookieInfo(username string) (cookieInfo *LoginResponse_Data_CookieInfo, err error) {
	err = localdb.RCoverTx(func(tx *buntdb.Tx) error {
		var err error
		key := localdb.BilibiliUserCookieInfoKey(username)
		var cookieStr string
		cookieStr, err = tx.Get(key)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(cookieStr), &cookieInfo)
		return err
	})
	return
}

func (c *StateManager) Start() error {
	for _, pattern := range []localdb.KeyPatternFunc{
		c.GroupConcernStateKey, c.CurrentLiveKey, c.FreshKey,
		c.UserInfoKey, c.UserStatKey, c.DynamicIdKey} {
		c.CreatePatternIndex(pattern, nil)
	}
	return c.StateManager.Start()
}

func NewStateManager() *StateManager {
	sm := &StateManager{}
	sm.extraKey = NewExtraKey()
	sm.StateManager = concern.NewStateManagerWithCustomKey(NewKeySet(), false)
	return sm
}
