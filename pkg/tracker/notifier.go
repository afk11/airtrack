package tracker

type ProjectAircraftUpdateListener interface {
	NewAircraft(p *Project, s *Sighting)
	UpdatedAircraft(p *Project, s *Sighting)
	LostAircraft(p *Project, s *Sighting)
}

type ProjectStatusListener interface {
	Activated(project *Project)
	Deactivated(project *Project)
}