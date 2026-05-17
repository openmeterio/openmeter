/**
* An interface that allows async iterable iteration both to completion and by page.
*/
export interface PagedAsyncIterableIterator<
TElement,
TPageResponse,
TPageSettings,
> {
/**
  * The next method, part of the iteration protocol
  */
next(): Promise<IteratorResult<TElement>>;
/**
  * The connection to the async iterator, part of the iteration protocol
  */
[Symbol.asyncIterator](): PagedAsyncIterableIterator<
  TElement,
  TPageResponse,
  TPageSettings
>;
/**
  * Return an AsyncIterableIterator that works a page at a time
  */
byPage: (
  settings?: TPageSettings,
) => AsyncIterableIterator<TPageResponse>;
}  /**
* An interface that describes how to communicate with the service.
*/
interface PagedResult<
TElement,
TPageResponse,
TPageSettings,
> {
/**
  * Extract the paged elements from the response. Only support array of elements.
  * @param response paged response
  * @returns TElement[]
  */
getElements: (response: TPageResponse) => TElement[];
/**
  * A method that returns a page of results.
  */
getPage: (
  nextLinkOrContinuationToken?: string,
  settings?: TPageSettings,
) => Promise<{ pagedResponse: TPageResponse; nextToken?: string } | undefined>;
/**
  * a function to implement the `byPage` method on the paged async iterator.
  */
byPage: (
  settings?: TPageSettings,
) => AsyncIterableIterator<TPageResponse>;
}/**
* Options for the paging helper
*/
export interface BuildPagedAsyncIteratorOptions<
TElement,
TPageResponse,
TPageSettings,
> {
getElements: (response: TPageResponse) => TElement[];
getPagedResponse: (nextToken?: string, settings?: TPageSettings) =>  Promise<{ pagedResponse: TPageResponse; nextToken?: string } | undefined>;
}/**
* Helper to paginate results in a generic way and return a PagedAsyncIterableIterator
*/
export function buildPagedAsyncIterator<
TElement,
TPageResponse,
TPageSettings
>(
options: BuildPagedAsyncIteratorOptions<TElement, TPageResponse, TPageSettings>,
): PagedAsyncIterableIterator<TElement, TPageResponse, TPageSettings> {
const pagedResult: PagedResult<TElement, TPageResponse, TPageSettings> = {
  getElements: options.getElements,
  getPage: options.getPagedResponse,
  byPage: (setting?: TPageSettings) => {
    return getPageAsyncIterator(pagedResult, { setting });
  },
};
const iter = getItemAsyncIterator<TElement, TPageResponse, TPageSettings>(pagedResult);
return {
  next() {
    return iter.next();
  },
  [Symbol.asyncIterator]() {
    return this;
  },
  byPage: pagedResult.byPage,
};
}

async function* getItemAsyncIterator<TElement, TPage, TPageSettings>(
pagedResult: PagedResult<TElement, TPage, TPageSettings>,
): AsyncIterableIterator<TElement> {
const pages = getPageAsyncIterator(pagedResult);
for await (const page of pages) {
  const results = pagedResult.getElements(page);
  yield* results;
}
}

async function* getPageAsyncIterator<TElement, TPageResponse, TPageSettings>(
pagedResult: PagedResult<TElement, TPageResponse, TPageSettings>,
options: {
  setting?: TPageSettings;
} = {},
): AsyncIterableIterator<TPageResponse> {
let response = await pagedResult.getPage(undefined, options.setting);
let results = response?.pagedResponse;
let nextToken = response?.nextToken;
if (!results) {
  return;
}
yield results;
while (nextToken) {
  response = await pagedResult.getPage(nextToken, options.setting);
  if (!response) {
    return;
  }
  results = response.pagedResponse;
  nextToken = response.nextToken;
  yield results;
}
}
