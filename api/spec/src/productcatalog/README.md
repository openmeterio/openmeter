# Plan Architecture

## API

### Endpoints

#### Plan

| Method | Endpoints                                                  | Description                          |
| ------ | ---------------------------------------------------------- | ------------------------------------ |
| GET    | /api/v1/plans                                              | List `Plan`s                         |
| POST   | /api/v1/plans                                              | Create a new `Plan`                  |
| GET    | /api/v1/plans/`:planId`                                    | Get `Plan`                           |
| DELETE | /api/v1/plans/`:planId`                                    | Delete `Plan`                        |
| PUT    | /api/v1/plans/`:planId`                                    | Update `Plan`                        |
| POST   | /api/v1/plans/`:planId`/publish                            | Publish `Plan`                       |
| POST   | /api/v1/plans/`:planId`/unpublish                          | Unpublish `Plan`                     |
| -      |                                                            | -                                    |
| GET    | /api/v1/plans/`:planKey`/versions                          | List all version of the `Plan`       |
| GET    | /api/v1/plans/`:planKey`/versions/`:planVersion`           | Get a specific version of the `Plan` |
| DELETE | /api/v1/plans/`:planKey`/versions/`:planVersion`           | Delete `Plan`                        |
| PUT    | /api/v1/plans/`:planKey`/versions/`:planVersion`           | Update `Plan`                        |
| POST   | /api/v1/plans/`:planKey`/versions/`:planVersion`/publish   | Publish `Plan`                       |
| POST   | /api/v1/plans/`:planKey`/versions/`:planVersion`/unpublish | Unpublish `Plan`                     |

`Plan` uniqueness constraints:
* `planId`
* `namespace` + `planKey` + `planVersion`

**TODO**

- [ ]: add endpoint for creating new version
- [ ]: add endpoint for creating a new plan from existing one


##### List plans

Request

```
GET /api/v1/plans?id=01J9RMXPP52YAVK744NECXS8NK&id=plan02
```

Response

```json5
{
  ...
  items: [
    {
      "id": "01J9RMXPP52YAVK744NECXS8NK",
      "key": "plan01",
      "version": 5,
      ...
    },
    {
      "id": "01J9RN1W4XGEVJKQPN56B5R9WM",
      "key": "plan02",
      "version": 2,
      ...
    },
  ]
}
```

##### Create plan

Request

```
POST /api/v1/plans
```

Response

```json5
{
  "key": "plan01",
  "currency": "USD",
  "phases": [
    {
      "key": "phase01",
      ...
    },
     {
      "key": "phase02",
      ...
    },
     {
      "key": "phase03",
      ...
    }
  ]
}
```

#### Phases

| Method  | Endpoints                                                           | Description                                |
| ------- | ------------------------------------------------------------------- | ------------------------------------------ |
| GET     | /api/v1/plans/`:planId`/phases                                      | List all available `PlanPhase`s for `Plan` |
| POST*   | /api/v1/plans/`:planId`/phases/`:phaseKey`                          | Create new `PlanPhase` for `Plan`          |
| GET     | /api/v1/plans/`:planId`/phases/`:phaseKey`                          | Create `PlanPhase` for `Plan`              |
| DELETE* | /api/v1/plans/`:planId`/phases/`:phaseKey`                          | Delete a `PlanPhase` in `Plan`             |
| -       | -                                                                   | -                                          |
| GET     | /api/v1/plans/`:planKey`/versions/`:planVersion`/phases             | List all available `PlanPhase`s for `Plan` |
| POST*   | /api/v1/plans/`:planKey`/versions/`:planVersion`/phases/`:phaseKey` | Create new `PlanPhase` for `Plan`          |
| GET     | /api/v1/plans/`:planKey`/versions/`:planVersion`/phases/`:phaseKey` | Get `PlanPhase` for `Plan`                 |
| DELETE* | /api/v1/plans/`:planKey`/versions/`:planVersion`/phases/`:phaseKey` | Delete `PlanPhase` for `Plan`              |

`PlanPhase` uniqueness constraints:
* `planId` + `phaseKey`
* `namespace` + `planKey` + `planVersion` + `phaseKey`

## Open Questions
| Q                                                                                            | A   |
| -------------------------------------------------------------------------------------------- | --- |
| Do we want to allow adding/deleting `PlanPhases` directly on `Plan` via phases API endpoint? | [ ] |
| Do we want to have endpoint for validating `Plan`                                            | [ ] |
| How many draft versions a `Plan` can have? One or multiple?                                  | [ ] |
