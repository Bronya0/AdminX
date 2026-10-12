package service

import (
	"bytes"
	"encoding/json"
	"fmt"

	apperr "adminx/pkg/errors"
)

// decodeStrictJSON 解码导入文件：未知字段直接报错。
// 拼错的键不会被静默忽略，不被接受的字段（如字典导入里的 is_encrypted）也会明确拒绝。
func decodeStrictJSON(raw []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// checkSettingsFileHeader 校验文件类型与格式版本，防止把别的文件或新版本格式误导入。
func checkSettingsFileHeader(kind, wantKind string, version, wantVersion int) error {
	if kind != wantKind {
		return apperr.New(400, "文件类型不匹配（应为 "+wantKind+"）")
	}
	if version != wantVersion {
		return apperr.New(400, fmt.Sprintf("不支持的文件版本 %d（当前支持 %d）", version, wantVersion))
	}
	return nil
}
