ALTER TABLE `sighting`
    ADD INDEX `sighting_aircraft_session` (`aircraft_id`,`session_id`);