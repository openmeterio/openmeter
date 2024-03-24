export const getQueryMaterializedViewQuery = (viewName: string, valueSQL: string, whereSQL: string): string => `
SELECT
      tumbleStart(
        windowstart,
        toIntervalHour(1),
        'UTC'
      ) AS windowstart,
      tumbleEnd(
        windowstart,
        toIntervalHour(1),
        'UTC'
      ) AS windowend,
      ${valueSQL} AS value,
      subject,
      group1,
      group2
    FROM ${viewName}
    ${whereSQL}
    GROUP BY
      windowstart,
      windowend,
      subject,
      group1,
      group2
    ORDER BY
      windowstart
`;
