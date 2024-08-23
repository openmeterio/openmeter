# Seeding OpenMeter with sample data

It's often useful to seed OpenMeter with test data during development.

We use [Benthos](https://www.benthos.dev) as a seeder (even though it's not built for this purpose, it works just fine).

To run the seeder continuously ingesting data, run the following command:

```sh
make seed
```

If you would like to see the generated events, enable logging:

```sh
make SEEDER_LOG=true seed
```

In some cases, you might want to fine tune the total number of events and at what rate they are ingested:

```sh
make SEEDER_COUNT=100 SEEDER_INTERVAL=1s seed
```

This tells Benthos to ingest 100 events at 1 event/s rate.
(By default Benthos will ingest 1 event / 50ms until the process is stopped)

Finally, you can configure the maximum number of subjects you would like to see:

```sh
make SEEDER_MAX_SUBJECTS=100 seed
```

(The default is 100)

Take a look at [seed.yaml](../etc/seed/seed.yaml) for further details.

You can find seeders for all meters defined in the example config. You can run them together via Benthos's streams mode:

```sh
benthos -c ./etc/seed/observability.yaml streams ./etc/seed/streams/*.yaml
```
