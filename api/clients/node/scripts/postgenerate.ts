/*
For all generated files, remove the ignore headers.
See related issue: https://github.com/ferdikoomen/openapi-typescript-codegen/issues/1539
*/

import { PathLike } from 'fs'
import fs from 'fs/promises'
import path from 'path'

const generatedDirUrl = new URL('../generated', import.meta.url)

const removeComments = [
	'/* istanbul ignore file */',
	'/* tslint:disable */',
	'/* eslint-disable */',
	'// @ts-ignore',
]

async function walk(dir: PathLike) {
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
			// write back
			await fs.writeFile(p, fileContent)
		}
		if (dirent.isDirectory()) {
			await walk(p)
		}
	}
}

await walk(generatedDirUrl)
