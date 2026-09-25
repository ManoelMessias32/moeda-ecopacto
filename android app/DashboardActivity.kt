package com.ecopacto.moeda

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.net.URL

// Modelos de dados para bater com o servidor Go
data class Tokenomics(
    val total_supply: Long,
    val game_rewards: Long,
    val dev_emergency: Long,
    val symbol: String
)

data class Block(
    val index: Int,
    val hash: String,
    val data: String
)

class DashboardActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            EcopactoDashboard()
        }
    }
}

@Composable
fun EcopactoDashboard() {
    var supplyInfo by remember { mutableStateOf("Carregando...") }
    var blocks by remember { mutableStateOf(listOf<Block>()) }

    // Efeito para buscar dados do servidor Go
    LaunchedEffect(Unit) {
        withContext(Dispatchers.IO) {
            try {
                // Aqui o App conecta no IP do seu computador onde o Go está rodando
                val response = URL("http://10.0.2.2:8080/tokenomics").readText()
                supplyInfo = response
            } catch (e: Exception) {
                supplyInfo = "Servidor Offline"
            }
        }
    }

    Scaffold(
        topBar = {
            SmallTopAppBar(title = { Text("Ecopacto Network Monitor") })
        }
    ) { padding ->
        Column(modifier = Modifier.padding(padding).padding(16.dp)) {
            Text("Monitor de Distribuição", fontSize = 20.sp, color = Color.Green)
            
            Spacer(modifier = Modifier.height(10.dp))
            
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Total Supply: 12 Trilhões ECO")
                    Text("Reserva Dev: 1 Trilhão ECO")
                    Text("Status da Rede: $supplyInfo")
                }
            }

            Spacer(modifier = Modifier.height(20.dp))
            Text("Últimos Blocos Minerados:", fontSize = 18.sp)
            
            LazyColumn {
                items(blocks) { block ->
                    Text("Bloco #${block.index}: ${block.hash.take(15)}...")
                }
            }
        }
    }
}
