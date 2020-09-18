create table `iaa_registry` (
    `id` integer not null primary key autoincrement,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `icao_hex` varchar(6) null,
    `registration` varchar(12) not null,
    `aircraft_manufacturer` varchar(255) null,
    `aircraft_model` varchar(255) null,
    `aircraft_category` varchar(255) null,
    `registration_date` datetime not null,
    `mtow` int null,
    `year_of_manufacture` int null,
    `aircraft_serial` varchar(255) null,
    `engine_manufacturer` varchar(255) null,
    `engine_model` varchar(255) null,
    `engine_number` int null,
    `owner_name` varchar(255) null,
    `owner_address` text null
                            );
create unique index iaa_registry_icao_hex_unique on iaa_registry(`icao_hex`);
