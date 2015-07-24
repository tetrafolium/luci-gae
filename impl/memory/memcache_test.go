// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"testing"
	"time"

	mcS "github.com/luci/gae/service/memcache"
	"github.com/luci/luci-go/common/clock/testclock"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

func TestMemcache(t *testing.T) {
	t.Parallel()

	Convey("memcache", t, func() {
		now := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
		c, tc := testclock.UseTime(context.Background(), now)
		c = Use(c)
		mc := mcS.Get(c)

		Convey("implements MCSingleReadWriter", func() {
			Convey("Add", func() {
				itm := &mcItem{
					key:        "sup",
					value:      []byte("cool"),
					expiration: time.Second,
				}
				err := mc.Add(itm)
				So(err, ShouldBeNil)
				Convey("which rejects objects already there", func() {
					err := mc.Add(itm)
					So(err, ShouldEqual, mcS.ErrNotStored)
				})
			})

			Convey("Get", func() {
				itm := &mcItem{
					key:        "sup",
					value:      []byte("cool"),
					expiration: time.Second,
				}
				err := mc.Add(itm)
				So(err, ShouldBeNil)

				testItem := &mcItem{
					key:   "sup",
					value: []byte("cool"),
					CasID: 1,
				}
				i, err := mc.Get("sup")
				So(err, ShouldBeNil)
				So(i, ShouldResemble, testItem)

				Convey("which can expire", func() {
					tc.Add(time.Second * 4)
					i, err := mc.Get("sup")
					So(err, ShouldEqual, mcS.ErrCacheMiss)
					So(i, ShouldBeNil)
				})
			})

			Convey("Delete", func() {
				Convey("works if it's there", func() {
					itm := &mcItem{
						key:        "sup",
						value:      []byte("cool"),
						expiration: time.Second,
					}
					err := mc.Add(itm)
					So(err, ShouldBeNil)

					err = mc.Delete("sup")
					So(err, ShouldBeNil)

					i, err := mc.Get("sup")
					So(err, ShouldEqual, mcS.ErrCacheMiss)
					So(i, ShouldBeNil)
				})

				Convey("but not if it's not there", func() {
					err := mc.Delete("sup")
					So(err, ShouldEqual, mcS.ErrCacheMiss)
				})
			})

			Convey("Set", func() {
				itm := &mcItem{
					key:        "sup",
					value:      []byte("cool"),
					expiration: time.Second,
				}
				err := mc.Add(itm)
				So(err, ShouldBeNil)

				itm.SetValue([]byte("newp"))
				err = mc.Set(itm)
				So(err, ShouldBeNil)

				testItem := &mcItem{
					key:   "sup",
					value: []byte("newp"),
					CasID: 2,
				}
				i, err := mc.Get("sup")
				So(err, ShouldBeNil)
				So(i, ShouldResemble, testItem)
			})

			Convey("CompareAndSwap", func() {
				itm := mcS.Item(&mcItem{
					key:        "sup",
					value:      []byte("cool"),
					expiration: time.Second * 2,
				})
				err := mc.Add(itm)
				So(err, ShouldBeNil)

				Convey("works after a Get", func() {
					itm, err = mc.Get("sup")
					So(err, ShouldBeNil)
					So(itm.(*mcItem).CasID, ShouldEqual, 1)

					itm.SetValue([]byte("newp"))
					err = mc.CompareAndSwap(itm)
					So(err, ShouldBeNil)
				})

				Convey("but fails if you don't", func() {
					itm.SetValue([]byte("newp"))
					err = mc.CompareAndSwap(itm)
					So(err, ShouldEqual, mcS.ErrCASConflict)
				})

				Convey("and fails if the item is expired/gone", func() {
					tc.Add(3 * time.Second)
					itm.SetValue([]byte("newp"))
					err = mc.CompareAndSwap(itm)
					So(err, ShouldEqual, mcS.ErrNotStored)
				})
			})
		})

		Convey("check that the internal implementation is sane", func() {
			curTime := now
			err := mc.Add(&mcItem{
				key:        "sup",
				value:      []byte("cool"),
				expiration: time.Second * 2,
			})

			mci := mc.(*memcacheImpl)

			So(err, ShouldBeNil)
			So(len(mci.data.items), ShouldEqual, 1)
			So(mci.data.casID, ShouldEqual, 1)
			So(mci.data.items["sup"], ShouldResemble, &mcItem{
				key:        "sup",
				value:      []byte("cool"),
				expiration: time.Duration(curTime.Add(time.Second * 2).UnixNano()),
				CasID:      1,
			})

			el, err := mc.Get("sup")
			So(err, ShouldBeNil)
			So(len(mci.data.items), ShouldEqual, 1)
			So(mci.data.casID, ShouldEqual, 1)

			testItem := &mcItem{
				key:   "sup",
				value: []byte("cool"),
				CasID: 1,
			}
			So(el, ShouldResemble, testItem)
		})

	})
}