package passkey

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
)

var errSessionNotFound = errors.New("Passkey 会话不存在或已过期")

func SaveSessionData(c *gin.Context, key string, data *webauthn.SessionData) error {
	session := sessions.Default(c)
	if data == nil {
		session.Delete(key)
		session.Delete(key + "_proof")
		return session.Save()
	}
	payload, err := common.Marshal(data)
	if err != nil {
		return err
	}
	userID, _ := session.Get("id").(int)
	proof, err := model.IssueVerificationProof(userID, "passkey:"+key, 5*time.Minute)
	if err != nil {
		return err
	}
	session.Set(key, string(payload))
	session.Set(key+"_proof", proof)
	return session.Save()
}

func PopSessionData(c *gin.Context, key string) (*webauthn.SessionData, error) {
	session := sessions.Default(c)
	raw := session.Get(key)
	if raw == nil {
		return nil, errSessionNotFound
	}
	session.Delete(key)
	proof, _ := session.Get(key + "_proof").(string)
	session.Delete(key + "_proof")
	if err := session.Save(); err != nil {
		return nil, err
	}
	userID, _ := session.Get("id").(int)
	valid, err := model.ConsumeVerificationProof(proof, userID, "passkey:"+key)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errSessionNotFound
	}
	var data webauthn.SessionData
	switch value := raw.(type) {
	case string:
		if err := common.Unmarshal([]byte(value), &data); err != nil {
			return nil, err
		}
	case []byte:
		if err := common.Unmarshal(value, &data); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("Passkey 会话格式无效")
	}
	return &data, nil
}
