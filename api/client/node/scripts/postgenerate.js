/*
For all generated files, remove the ignore headers.
See related issue: https://github.com/ferdikoomen/openapi-typescript-codegen/issues/1539
*/

import fs from 'fs/promises'
import path from 'path'

const generatedDirUrl = new URL('../generated', import.meta.url)

const removeComments = [
	'/* istanbul ignore file */',
	'/* tslint:disable */',
	'/* eslint-disable */',
	'// @ts-ignore',
]

/**
 *
 * @param {import('fs').PathLike} dir
 */
async function walk(dir) {
	const dirents = await fs.readdir(dir, { withFileTypes: true })
	for (const dirent of dirents) {
		const p = path.join(dir.toString(), dirent.name).replace('file:', '')
		if (dirent.isFile()) {
			const f = await fs.readFile(p, 'utf-8')
			// remove headers
			let fileContent = f
			for (const c of removeComments) {
				fileContent = fileContent.replace(c, '')
			}
			// TODO: generated code type output for succesfull response is incorrect
			// For example it generates `CancelablePromise<Meter | Error>` instead of `CancelablePromise<Meter>
			// The library behavior is correct and the promise throws in the case of error.
			if (dirent.name === 'DefaultService.ts') {
				fileContent = fileContent.replace(/ \| Error/g, '')
			}
			// write back
			await fs.writeFile(p, fileContent)
		}
		if (dirent.isDirectory()) {
			await walk(p)
		}
	}
}

await walk(generatedDirUrl)
