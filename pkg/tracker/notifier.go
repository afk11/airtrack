package tracker

// ProjectAircraftUpdateListener - this interface is used to
// communicate information about a projects aircraft sightings
type ProjectAircraftUpdateListener interface {
	// NewAircraft informs listener a new sighting was opened for a project
	NewAircraft(p *Project, s *Sighting)
	// UpdatedAircraft informs listener about an updated aircraft for a project
	UpdatedAircraft(p *Project, s *Sighting)
	// LostAircraft informs listener about a sighting which has closed for a project
	LostAircraft(p *Project, s *Sighting)
}

// ProjectStatusListener - this interface is used to
// communicate status changes about a project
type ProjectStatusListener interface {
	// Activate informs listener a new project was activated
	Activated(project *Project)
	// Deactivated informs listener a project was deactivated
	Deactivated(project *Project)
}
