<script setup>
import { ref } from 'vue'
import axios from '../services/axios.js'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const errormsg = ref(null)
const loading = ref(false)

async function doLogin() {
    loading.value = true
    errormsg.value = null
    try {
        // 1. Chiamiamo il backend per fare login
        let response = await axios.post('/session', { name: username.value })
        
        // 2. IMPORTANTISSIMO: Salviamo il token (identifier) che ci da il server
        // Se il tuo backend restituisce un JSON tipo { "identifier": "12345" }
        // Assicurati che response.data.identifier sia corretto.
        // Se il tuo backend restituisce solo il testo ID, usa response.data
        if (response.data.identifier) {
            localStorage.setItem('token', response.data.identifier)
        } else {
             // Fallback se il tuo backend restituisce l'ID in un altro modo
            localStorage.setItem('token', response.data)
        }
        
        localStorage.setItem('username', username.value)

        // 3. Andiamo alla home
        router.push('/') 
    } catch (e) {
        if (e.response && e.response.data) {
            errormsg.value = e.response.data.message || JSON.stringify(e.response.data)
        } else {
            errormsg.value = e.toString()
        }
    }
    loading.value = false
}
</script>

<template>
    <div class="d-flex align-items-center justify-content-center vh-100">
        <div class="card p-4 shadow-sm" style="width: 350px;">
            <h3 class="mb-3 text-center">Login Wasa</h3>
            
            <div class="mb-3">
                <label for="username" class="form-label">Username</label>
                <input 
                    type="text" 
                    class="form-control" 
                    id="username" 
                    v-model="username" 
                    placeholder="Scegli un nome utente"
                    @keyup.enter="doLogin"
                >
            </div>

            <button 
                type="button" 
                class="btn btn-primary w-100" 
                @click="doLogin"
                :disabled="loading"
            >
                <span v-if="loading" class="spinner-border spinner-border-sm me-2"></span>
                {{ loading ? 'Accesso in corso...' : 'Entra' }}
            </button>

            <div v-if="errormsg" class="alert alert-danger mt-3 small p-2">
                {{ errormsg }}
            </div>
        </div>
    </div>
</template>

<style>
.vh-100 { min-height: 100vh; }
</style>