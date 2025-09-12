import { createInertiaApp } from '@inertiajs/svelte'
import { mount } from 'svelte'
import './app.css'

createInertiaApp({
	id: 'app',
	resolve: name => {
		const pages = import.meta.glob('./pages/**/*.svelte', { eager: true })
		return pages[`./pages/${name}.svelte`]
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
