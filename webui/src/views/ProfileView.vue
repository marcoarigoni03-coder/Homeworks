<script setup>
import { ref, onMounted } from 'vue'
import axios from '../services/axios.js'

const username = ref(localStorage.getItem('username') || '')
const newName = ref('')
const msg = ref(null)

async function updateName() {
    if (!newName.value) return
    try {
        let token = localStorage.getItem('token')
        // Chiamata PUT per aggiornare il nome
        await axios.put('/users/me/name', 
            { name: newName.value }, 
            { headers: { Authorization: token } }
        )
        
        // Aggiorniamo i dati locali
        username.value = newName.value
        localStorage.setItem('username', newName.value)
        newName.value = ''
        msg.value = "Nome aggiornato con successo!"
    } catch (e) {
        msg.value = "Errore: " + (e.response?.data || e.toString())
    }
}
</script>

<template>
    <div class="container mt-5">
        <div class="card shadow-sm">
            <div class="card-header bg-primary text-white">
                <h4>Il mio Profilo</h4>
            </div>
            <div class="card-body">
                <p><strong>Username attuale:</strong> {{ username }}</p>
                <hr>
                
                <div class="mb-3">
                    <label class="form-label">Cambia il tuo nome:</label>
                    <div class="input-group">
                        <input type="text" class="form-control" v-model="newName" placeholder="Nuovo nome">
                        <button class="btn btn-success" @click="updateName">Salva</button>
                    </div>
                </div>

                <div v-if="msg" class="alert alert-info mt-2">
                    {{ msg }}
                </div>
            </div>
        </div>
        
        <div class="mt-3">
            <router-link to="/" class="btn btn-outline-secondary">← Torna alla Home</router-link>
        </div>
    </div>
</template>