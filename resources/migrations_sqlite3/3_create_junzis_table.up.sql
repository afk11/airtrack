create table `junzis_registry` (
    `id` integer not null primary key autoincrement,
     `created_at` timestamp null,
     `updated_at` timestamp null,
     `icao_hex` varchar(6) not null,
     `registration` varchar(12) not null,
     `aircraft_manufacturer` varchar(255) null,
     `aircraft_model` varchar(255) not null,
     `owner_name` varchar(255) null
);
create unique index junzis_registry_icao_hex_unique on junzis_registry(`icao_hex`);
create index junzis_registry_registration_index on junzis_registry(`registration`);