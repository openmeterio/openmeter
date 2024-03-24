# Running Performance Tests

Performance testing & benchmarking is a complicated topic that's hard to get right with synthetic environments.

We have some comparative benchmarks written in k6 under `perf/k6`. The measured numbers are highly dependent on the environment and the hardware they are run on. The numbers are not absolute, but they can give you a rough idea of performance.

## Running the tests

As the test results are highly dependent on the environment, the tests can be run separatly from any environment setup automations to allow for more controlled testing / further experimentation. Running on your local, the environments setup would consist of:

1. Running the OpenMeter stack. For the tests to work the meters defined in `perf/configs/config.json` must exist. The simplest way to achieve this is to run OpenMeter with that config file.
2. Seeding OpenMeter. `perf/configs/seed.benthos.yaml` generates data so that it matches the defined meters. Feel free to alter the SEEDER_COUNT to your liking, the default is 1M.
3. Running the tests. Check the `perf/k6` folder for the different options on how to run them.

Alternatively you can run the tests against a containerised configuration with dagger (`dagger call perf`).
