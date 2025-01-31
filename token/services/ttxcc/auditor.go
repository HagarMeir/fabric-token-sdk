/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc"
)

type txAuditor struct {
	auditor *auditor.Auditor
}

func NewAuditor(sp view2.ServiceProvider, w *token.AuditorWallet) *txAuditor {
	return &txAuditor{
		auditor: auditor.New(sp, w),
	}
}

func (a *txAuditor) Validate(tx *Transaction) error {
	return a.auditor.Validate(tx.TokenRequest)
}

func (a *txAuditor) Audit(tx *Transaction) (*token.InputStream, *token.OutputStream, error) {
	return a.auditor.Audit(tx.TokenRequest)
}

func (a *txAuditor) NewQueryExecutor() *auditor.QueryExecutor {
	return a.auditor.NewQueryExecutor()
}

type RegisterAuditorView struct {
	Network   string
	Channel   string
	Namespace string
	Id        view.Identity
	AuditView view.View
}

func NewRegisterAuditorView(id view.Identity, auditView view.View) *RegisterAuditorView {
	return &RegisterAuditorView{Id: id, AuditView: auditView}
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	view2.GetRegistry(context).RegisterResponder(r.AuditView, &AuditingViewInitiator{})

	return context.RunView(tcc.NewRegisterAuditorView(r.Network, r.Channel, r.Namespace, r.Id))
}

type AuditingViewInitiator struct {
	tx *Transaction
}

func newAuditingViewInitiator(tx *Transaction) *AuditingViewInitiator {
	return &AuditingViewInitiator{tx: tx}
}

func (a *AuditingViewInitiator) Call(context view.Context) (interface{}, error) {
	session, err := context.GetSession(a, a.tx.opts.auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting session")
	}

	// Send transaction
	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, err
	}
	err = session.Send(txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	// Receive signature
	ch := session.Receive()
	var msg *view.Message
	select {
	case msg = <-ch:
		logger.Debug("reply received from %s", a.tx.opts.auditor)
	case <-time.After(60 * time.Second):
		return nil, errors.Errorf("Timeout from party %s", a.tx.opts.auditor)
	}
	if msg.Status == view.ERROR {
		return nil, errors.New(string(msg.Payload))
	}

	// TODO: IsValid it?

	// Check signature
	signed, err := a.tx.MarshallToAudit()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling message to sign")
	}
	logger.Debugf("Verifying auditor signature on [%s][%s][%s]", a.tx.opts.auditor.UniqueID(), hash.Hashable(signed).String(), a.tx.ID())

	v, err := a.tx.TokenService().SigService().GetVerifier(a.tx.opts.auditor)
	if err != nil {
		return nil, err
	}
	if err := v.Verify(signed, msg.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed verifying auditor signature")
	}

	a.tx.TokenRequest.SetAuditorSignature(msg.Payload)

	return nil, nil
}

type AuditApproveView struct {
	w  *token.AuditorWallet
	tx *Transaction
}

func NewAuditApproveView(w *token.AuditorWallet, tx *Transaction) *AuditApproveView {
	return &AuditApproveView{w: w, tx: tx}
}

func (a *AuditApproveView) Call(context view.Context) (interface{}, error) {
	// Append audit records
	logger.Debugf("store audit records...")
	auditRecord, err := a.tx.TokenRequest.AuditRecord()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting audit records for tx [%s]", a.tx.ID())
	}
	if err := auditdb.GetAuditDB(context, a.w).Append(auditRecord); err != nil {
		return nil, errors.WithMessagef(err, "failed appening audit records for tx [%s]", a.tx.ID())
	}
	logger.Debugf("store audit records...done")

	logger.Debugf("sign and send back")
	if err := a.signAndSendBack(context); err != nil {
		return nil, err
	}

	logger.Debugf("audit approve done")
	return nil, nil
}

func (a *AuditApproveView) signAndSendBack(context view.Context) error {
	// Sign
	aid, err := a.w.GetAuditorIdentity()
	if err != nil {
		return errors.WithMessagef(err, "failed getting auditor identity for [%s]", context.Me())
	}
	signer, err := a.w.GetSigner(aid)
	if err != nil {
		return errors.WithMessagef(err, "failed getting signing identity for auditor identity [%s]", context.Me())
	}

	raw, err := a.tx.MarshallToAudit()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling tx [%s] to audit", a.tx.ID())
	}

	logger.Debugf("Endorse [%s][%s][%s]", aid.UniqueID(), hash.Hashable(raw).String(), a.tx.ID())
	sigma, err := signer.Sign(raw)
	if err != nil {
		return errors.Wrapf(err, "failed sign audit message for tx [%s]", a.tx.ID())
	}

	session := context.Session()
	if err := session.Send(sigma); err != nil {
		return errors.WithMessagef(err, "failed sending back auditor signature")
	}

	if err := a.waitFabricEnvelope(context); err != nil {
		return errors.WithMessagef(err, "failed obtaining auditor signature")
	}
	return nil
}

func (a *AuditApproveView) waitFabricEnvelope(context view.Context) error {
	tx, err := ReceiveTransaction(context)
	if err != nil {
		return errors.Wrapf(err, "failed receiving transaction")
	}

	// Processes
	logger.Debugf("Processes Fabric Envelope...")
	env := tx.Payload.FabricEnvelope
	if env == nil {
		return errors.Errorf("expected fabric envelope")
	}

	err = tx.storeTransient()
	if err != nil {
		return errors.Wrapf(err, "failed storing transient")
	}

	ch := fabric.GetChannel(context, tx.Network(), tx.Channel())
	rws, err := ch.Vault().GetRWSet(tx.ID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset for tx [%s]", tx.ID())
	}
	rws.Done()

	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", tx.ID())
	}
	if err := ch.Vault().StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}

	// Send the proposal response back
	logger.Debugf("Send the ack")
	err = context.Session().Send([]byte("ack"))
	if err != nil {
		return err
	}

	return nil
}
