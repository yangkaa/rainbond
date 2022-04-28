// Copyright (C) 2014-2018 Goodrain Co., Ltd.
// RAINBOND, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package db

import "testing"

//func TestFileSaveMessage(t *testing.T) {
//
//	f := filePlugin{
//		homePath: "./test",
//	}
//	m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
//	//m.Content = []byte("----------")
//	mes := []*EventLogMessage{m}
//	//for i := 0; i < 99999; i++ {
//	//	m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
//	//	switch {
//	//	case i < 9999:
//	//		m.Content = []byte("1111111111")
//	//	case i >= 9999 && i < 19999:
//	//		m.Content = []byte("2222222222")
//	//	case i >= 19999 && i < 29999:
//	//		m.Content = []byte("3333333333")
//	//	case i >= 29999 && i < 39999:
//	//		m.Content = []byte("4444444444")
//	//	case i >= 39999 && i < 49999:
//	//		m.Content = []byte("5555555555")
//	//	case i >= 49999 && i < 59999:
//	//		m.Content = []byte("6666666666")
//	//	case i >= 59999 && i < 69999:
//	//		m.Content = []byte("7777777777")
//	//	case i >= 69999 && i < 79999:
//	//		m.Content = []byte("8888888888")
//	//	case i >= 79999 && i < 89999:
//	//		m.Content = []byte("9999999999")
//	//	case i >= 89999 && i < 99999:
//	//		m.Content = []byte("0000000000")
//	//	}
//	//	//m.Content = []byte("1234567890")
//	//	mes = append(mes, m)
//	//}
//
//	for i := 0; i < 15000; i++ {
//		m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
//		m.Content = []byte("1234567890")
//		mes = append(mes, m)
//	}
//
//	err := f.SaveMessage(mes)
//	if err != nil {
//		t.Fatal(err)
//	}
//}

func TestFileSaveMessage(t *testing.T) {
	//content := "1234567890"
	//content := "2222222222"
	content := "3333333333"
	//content := "1111111111"
	count := 49
	f := filePlugin{
		homePath: "./test",
	}
	m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl", Content: []byte(content)}
	mes := []*EventLogMessage{m}

	for i := 0; i < count; i++ {
		m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
		m.Content = []byte(content)
		//m.Content = []byte("2222222222")
		//m.Content = []byte("3333333333")
		//m.Content = []byte("4444444444")
		mes = append(mes, m)
	}

	err := f.SaveMessage(mes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMvLogFile(t *testing.T) {
	MvLogFile("/Users/qingguo/7b3d5546bd54152d/stdout.log.gz", "/Users/qingguo/7b3d5546bd54152d/stdout.log")
}

func TestGetMessages(t *testing.T) {
	f := filePlugin{
		homePath: "./test",
	}
	logs, err := f.GetMessages("qwertyuiopasdfghjkl", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(logs)
	logs, err = f.GetMessages("qwertyuiopasdfghjkl", "", -10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(logs)
}
