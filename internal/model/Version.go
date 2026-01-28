package model

import (
    "errors"
    "reflect"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type Versioned struct {
    Version int `gorm:"column:version;default:1"`
}

func (v *Versioned) BeforeUpdate(tx *gorm.DB) error {
    if tx == nil || tx.Statement == nil {
        return nil
    }

    // 如果没有可反射的值（例如使用 map 更新或批量更新），跳过 Hook。
    rv := tx.Statement.ReflectValue
    if !rv.IsValid() {
        return nil
    }
    if rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct {
        return nil
    }

    // 尝试从当前结构体中读取 Version 字段的实际值（比直接用 v.Version 更可靠）
    fv := rv.FieldByName("Version")
    if !fv.IsValid() || !fv.CanInterface() {
        return nil
    }

    var ver int
    switch fv.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        ver = int(fv.Int())
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        ver = int(fv.Uint())
    default:
        // 非整数类型，跳过
        return nil
    }

    // 获取 DB 列名，优先使用 schema 信息
    colName := "version"
    if tx.Statement.Schema != nil {
        if f := tx.Statement.Schema.LookUpField("Version"); f != nil {
            colName = f.DBName
        }
    }

    // 注入 WHERE version = ?
    tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{
        clause.Eq{Column: clause.Column{Name: colName}, Value: ver},
    }})

    // 设置要写入的新版本号（SetColumn 会把字段注入到更新语句中）
    tx.Statement.SetColumn("Version", ver+1)

    return nil
}

func (v *Versioned) AfterUpdate(tx *gorm.DB) error {
    // 只有当是基于结构体实例的更新且包含 Version 字段时才检查
    if tx == nil || tx.Statement == nil {
        return nil
    }

    rv := tx.Statement.ReflectValue
    if !rv.IsValid() {
        return nil
    }
    if rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct {
        return nil
    }
    if _, ok := rv.Type().FieldByName("Version"); !ok {
        return nil
    }

    if tx.RowsAffected == 0 {
        // 标记为错误，调用方会收到该错误
        return errors.New("optimistic lock: record has been modified by another transaction")
    }
    return nil
}