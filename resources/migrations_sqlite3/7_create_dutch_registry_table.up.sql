create table `dutch_registry` (
    `id` integer not null primary key autoincrement,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `icao_hex` varchar(6) null,
    `registration` varchar(12) not null,
    `registration_number` int not null,
    `aircraft_model` varchar(255) null,
    `registration_date` datetime null,
    `mtow` int null,
    `year_of_manufacture` int null,
    `owner_name` varchar(255) null,
    `owner_address` text null
                              );
create unique index dutch_registry_icao_hex_unique on dutch_registry(`icao_hex`);
create unique index dutch_registry_registration_index on dutch_registry(`registration`);
