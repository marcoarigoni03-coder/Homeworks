<script setup>
import { ref, onMounted } from 'vue'
import axios from '../services/axios.js'

const conversations = ref([])
const loading = ref(false)
const errormsg = ref(null)

// Recuperiamo il token salvato nel login
const token = localStorage.getItem('token')

async function getConversations() {
    loading.value = true
    errormsg.value = null
    try {
        // CHIAMATA CORRETTA: Chiama la rotta definita nel tuo api.go
        // Aggiungiamo l'header Authorization perché il backend deve sapere chi sei
        let response = await axios.get('/conversations', {
            headers: {
                Authorization: token // O "Bearer " + token, dipende da come hai fatto l'auth.go
            }
        })
        
        conversations.value = response.data 
    } catch (e) {
        errormsg.value = e.response ? e.response.data : e.toString()
    }
    loading.value = false
}

onMounted(() => {
    getConversations()
})
</script>

<template>
    <div class="container mt-4">
        
        <div class="d-flex justify-content-between align-items-center mb-4">
            <h2>Le mie Conversazioni</h2>
            
            <router-link to="/profile" class="btn btn-primary">
                Il mio Profilo
            </router-link>
        </div>

        <div v-if="loading" class="text-center">
            <div class="spinner-border text-primary" role="status"></div>
        </div>

        <div v-if="errormsg" class="alert alert-danger">
            Errore caricamento: {{ errormsg }}
        </div>

        <div v-if="conversations.length === 0 && !loading" class="text-center text-muted mt-5">
            <p>Non hai ancora nessuna conversazione attiva.</p>
            <p>Per iniziare, devi cercare un utente (funzionalità da implementare).</p>
        </div>

        <div class="list-group">
            <button 
                v-for="chat in conversations" 
                :key="chat.id" 
                class="list-group-item list-group-item-action d-flex justify-content-between align-items-center"
            >
                <div>
                    <h5 class="mb-1">{{ chat.name || 'Conversazione ' + chat.id }}</h5>
                    <small class="text-muted">Ultimo messaggio...</small>
                </div>
                <span class="badge bg-primary rounded-pill">Apri</span>
            </button>
        </div>
    </div>
</template>

<style scoped>
</style>