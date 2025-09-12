import { createInertiaApp } from '@inertiajs/svelte'
import { mount } from 'svelte'
import './app.css'

createInertiaApp({
	id: 'app',
	resolve: async name => {
		try {
			const modules = import.meta.glob('./pages/**/*.svelte')
			const componentPath = `./pages/${name}.svelte`
			
			if (!modules[componentPath]) {
				console.error(`Component not found: ${name}`)
				console.log('Available components:', Object.keys(modules))
				throw new Error(`Component ${name} not found`)
			}
			
			const module = await modules[componentPath]()
			return module.default
		} catch (error) {
			console.error('Error loading component:', error)
			throw error
		}
	},
	setup({ el, App, props }) {
		mount(App, { target: el, props })
	},
	progress: { 
		delay: 250,
		color: '#ff6b35',
		includeCSS: true,
		showSpinner: true 
	}
})
