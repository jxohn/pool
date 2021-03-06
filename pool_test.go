package pool

import (
	"context"
	"database/sql"
	"net"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/smartystreets/goconvey/convey"
)

func TestNewPool(t *testing.T) {
	convey.Convey("test new pool", t, func() {
		convey.Convey("db pool", func() {
			db, err := sql.Open("mysql", "username:pwd@tcp(127.0.0.1:3306)/test")

			convey.So(err, convey.ShouldBeNil)

			producer := &DBProducer{
				DB: db,
			}

			pool, err := NewPool(10, 20, producer)
			convey.So(err, convey.ShouldBeNil)

			holdMap := make(map[Hold]bool)
			// take one hundred times synchronize
			for i := 0; i < 100; i++ {
				one, err := pool.Get()
				convey.So(err, convey.ShouldBeNil)
				hold := one.(*ProxyHold)
				holdMap[hold.Hold] = true
				err = one.Close()
				convey.So(err, convey.ShouldBeNil)
			}
			convey.So(len(holdMap), convey.ShouldEqual, 10)
			pool.Close()
		})

		convey.Convey("tcp pool", func() {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			convey.So(err, convey.ShouldBeNil)
			go listener.Accept()

			tcpProducer := &TcpProducer{
				Addr: listener.Addr().String(),
			}

			pool, err := NewPool(10, 20, tcpProducer)
			convey.So(err, convey.ShouldBeNil)

			holdMap := make(map[Hold]bool)
			for i := 0; i < 100; i++ {
				one, err := pool.Get()
				convey.So(err, convey.ShouldBeNil)
				hold := one.(*ProxyHold)
				holdMap[hold.Hold] = true
				err = one.Close()
				convey.So(err, convey.ShouldBeNil)
			}
			convey.So(len(holdMap), convey.ShouldEqual, 10)
			pool.Close()
		})
	})

}

type DBProducer struct {
	*sql.DB
}

func (d *DBProducer) Produce() (one Hold, err error) {
	return d.Conn(context.Background())
}

type TcpProducer struct {
	Addr string
}

func (t *TcpProducer) Produce() (one Hold, err error) {
	return net.Dial("tcp", t.Addr)
}
