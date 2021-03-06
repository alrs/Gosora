package common

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"

	qgen "github.com/Azareal/Gosora/query_gen"
)

var SettingBox atomic.Value // An atomic value pointing to a SettingBox

// SettingMap is a map type specifically for holding the various settings admins set to toggle features on and off or to otherwise alter Gosora's behaviour from the Control Panel
type SettingMap map[string]interface{}

type SettingStore interface {
	ParseSetting(sname, scontent, stype, sconstraint string) string
	BypassGet(name string) (*Setting, error)
	BypassGetAll(name string) ([]*Setting, error)
}

type OptionLabel struct {
	Label    string
	Value    int
	Selected bool
}

type Setting struct {
	Name       string
	Content    string
	Type       string
	Constraint string
}

type SettingStmts struct {
	getAll *sql.Stmt
	get    *sql.Stmt
	update *sql.Stmt
}

var settingStmts SettingStmts

func init() {
	SettingBox.Store(SettingMap(make(map[string]interface{})))
	DbInits.Add(func(acc *qgen.Accumulator) error {
		s := "settings"
		settingStmts = SettingStmts{
			getAll: acc.Select(s).Columns("name,content,type,constraints").Prepare(),
			get:    acc.Select(s).Columns("content,type,constraints").Where("name=?").Prepare(),
			update: acc.Update(s).Set("content=?").Where("name=?").Prepare(),
		}
		return acc.FirstError()
	})
}

func (s *Setting) Copy() (out *Setting) {
	out = &Setting{Name: ""}
	*out = *s
	return out
}

func LoadSettings() error {
	sBox := SettingMap(make(map[string]interface{}))
	settings, err := sBox.BypassGetAll()
	if err != nil {
		return err
	}

	for _, s := range settings {
		err = sBox.ParseSetting(s.Name, s.Content, s.Type, s.Constraint)
		if err != nil {
			return err
		}
	}

	SettingBox.Store(sBox)
	return nil
}

// TODO: Add better support for HTML attributes (html-attribute). E.g. Meta descriptions.
func (sBox SettingMap) ParseSetting(sname, scontent, stype, constraint string) (err error) {
	ssBox := map[string]interface{}(sBox)
	switch stype {
	case "bool":
		ssBox[sname] = (scontent == "1")
	case "int":
		ssBox[sname], err = strconv.Atoi(scontent)
		if err != nil {
			return errors.New("You were supposed to enter an integer x.x")
		}
	case "int64":
		ssBox[sname], err = strconv.ParseInt(scontent, 10, 64)
		if err != nil {
			return errors.New("You were supposed to enter an integer x.x")
		}
	case "list":
		cons := strings.Split(constraint, "-")
		if len(cons) < 2 {
			return errors.New("Invalid constraint! The second field wasn't set!")
		}

		con1, err := strconv.Atoi(cons[0])
		con2, err2 := strconv.Atoi(cons[1])
		if err != nil || err2 != nil {
			return errors.New("Invalid contraint! The constraint field wasn't an integer!")
		}

		value, err := strconv.Atoi(scontent)
		if err != nil {
			return errors.New("Only integers are allowed in this setting x.x")
		}

		if value < con1 || value > con2 {
			return errors.New("Only integers between a certain range are allowed in this setting")
		}
		ssBox[sname] = value
	default:
		ssBox[sname] = scontent
	}
	return nil
}

func (sBox SettingMap) BypassGet(name string) (*Setting, error) {
	s := &Setting{Name: name}
	err := settingStmts.get.QueryRow(name).Scan(&s.Content, &s.Type, &s.Constraint)
	return s, err
}

func (sBox SettingMap) BypassGetAll() (settingList []*Setting, err error) {
	rows, err := settingStmts.getAll.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		s := &Setting{Name: ""}
		err := rows.Scan(&s.Name, &s.Content, &s.Type, &s.Constraint)
		if err != nil {
			return nil, err
		}
		settingList = append(settingList, s)
	}
	return settingList, rows.Err()
}

func (sBox SettingMap) Update(name, content string) RouteError {
	s, err := sBox.BypassGet(name)
	if err == ErrNoRows {
		return FromError(err)
	} else if err != nil {
		return SysError(err.Error())
	}

	// TODO: Why is this here and not in a common function?
	if s.Type == "bool" {
		if content == "on" || content == "1" {
			content = "1"
		} else {
			content = "0"
		}
	}

	err = sBox.ParseSetting(name, content, s.Type, s.Constraint)
	if err != nil {
		return FromError(err)
	}

	// TODO: Make this a method or function?
	_, err = settingStmts.update.Exec(content, name)
	if err != nil {
		return SysError(err.Error())
	}

	err = LoadSettings()
	if err != nil {
		return SysError(err.Error())
	}
	return nil
}
