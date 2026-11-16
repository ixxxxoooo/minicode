package claudecfg

import "strings"

func (s *Store) ensurePicks() {
	if s.Picks == nil {
		s.Picks = map[string]string{}
	}
}

// SetPick 记住某服务商在运行配置弹层里选中的模型(官方用 PickOfficial)。
func (s *Store) SetPick(providerID, model string) {
	model = strings.TrimSpace(model)
	if providerID == "" || model == "" {
		return
	}
	s.ensurePicks()
	s.Picks[providerID] = model
}

// ResolveProviderPick 读取 picks;不在已勾选清单里时退到第一项,不报错。
func (s Store) ResolveProviderPick(providerID string) string {
	p := s.Get(providerID)
	if p == nil {
		return ""
	}
	model := ""
	if s.Picks != nil {
		model = strings.TrimSpace(s.Picks[providerID])
	}
	if model != "" && p.HasModel(model) {
		return model
	}
	if len(p.Models) > 0 {
		return p.Models[0]
	}
	return ""
}

// SanitizePicks 清理 picks 里已失效的模型 id(取消勾选 / 删服务商之后调用)。
func (s *Store) SanitizePicks() {
	if len(s.Picks) == 0 {
		return
	}
	s.ensurePicks()
	for id, m := range s.Picks {
		if id == PickOfficial {
			continue
		}
		p := s.Get(id)
		if p == nil {
			delete(s.Picks, id)
			continue
		}
		m = strings.TrimSpace(m)
		if m == "" || !p.HasModel(m) {
			if len(p.Models) > 0 {
				s.Picks[id] = p.Models[0]
			} else {
				delete(s.Picks, id)
			}
		}
	}
}
