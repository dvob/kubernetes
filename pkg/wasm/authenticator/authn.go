package authenticator

import (
	"context"
	"errors"

	authv1 "k8s.io/api/authentication/v1"
	authn "k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
)

type TokenReviewFunc func(context.Context, *authv1.TokenReview) (*authv1.TokenReview, error)

var _ authn.Token = (*Authenticator)(nil)

type Authenticator struct {
	review       TokenReviewFunc
	implicitAuds authn.Audiences
}

func (m *Authenticator) AuthenticateToken(ctx context.Context, token string) (*authn.Response, bool, error) {
	wantAuds, checkAuds := authn.AudiencesFrom(ctx)

	req := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: wantAuds,
		},
	}

	resp, err := m.review(ctx, req)
	if err != nil {
		klog.ErrorS(err, "failed to run authentication")
		return nil, false, err
	}

	tr := resp

	var auds authn.Audiences
	if checkAuds {
		gotAuds := m.implicitAuds
		if len(tr.Status.Audiences) > 0 {
			gotAuds = tr.Status.Audiences
		}
		auds = wantAuds.Intersect(gotAuds)
		if len(auds) == 0 {
			klog.V(4).InfoS("no matching audiences", "want_auds", wantAuds, "got_auds", gotAuds)
			return nil, false, nil
		}
	}

	if !tr.Status.Authenticated {
		if tr.Status.Error != "" {
			return nil, false, errors.New(tr.Status.Error)
		}
		return nil, false, nil
	}

	u := &user.DefaultInfo{
		Name:   tr.Status.User.Username,
		UID:    tr.Status.User.UID,
		Groups: tr.Status.User.Groups,
	}
	for key, value := range tr.Status.User.Extra {
		u.Extra[key] = value
	}

	return &authn.Response{
		Audiences: auds,
		User:      u,
	}, true, nil
}
