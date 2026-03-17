package actor

type SystemActor struct {
	serviceName string
}

func NewSystemActor(serviceName string) *SystemActor {
	return &SystemActor{serviceName: serviceName}
}

func (s *SystemActor) ID() string          { return "system:" + s.serviceName }
func (s *SystemActor) Type() Type          { return TypeSystem }
func (s *SystemActor) DisplayName() string { return s.serviceName }
func (s *SystemActor) ServiceName() string { return s.serviceName }
func (s *SystemActor) Scope(key string) string { return "" }
