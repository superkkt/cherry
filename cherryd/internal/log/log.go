/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package log

type Logger interface {
	Debug(m string) (err error)
	Err(m string) (err error)
	Info(m string) (err error)
	Notice(m string) (err error)
	Warning(m string) (err error)
}
