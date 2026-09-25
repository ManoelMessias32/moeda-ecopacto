package com.ecopacto.moeda

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL

class WalletActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            EcopactoWalletTheme {
                WalletScreen()
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WalletScreen() {
    val scope = rememberCoroutineScope()
    var username by remember { mutableStateOf("") }
    var walletAddress by remember { mutableStateOf("Nenhuma carteira criada") }
    var balance by remember { mutableStateOf("0 ECO") }
    var isMining by remember { mutableStateOf(false) }
    var miningLog by remember { mutableStateOf(listOf<String>()) }

    var destAddress by remember { mutableStateOf("") }
    var transferAmount by remember { mutableStateOf("") }
    var txStatus by remember { mutableStateOf("") }

    LazyColumn(modifier = Modifier.padding(16.dp).fillMaxSize()) {
        item {
            Text("Carteira & Mineração Ecopacto", style = MaterialTheme.typography.headlineMedium)
            Spacer(modifier = Modifier.height(16.dp))

            TextField(
                value = username,
                onValueChange = { username = it },
                label = { Text("Nome de Usuário") },
                modifier = Modifier.fillMaxWidth()
            )

            Button(
                onClick = {
                    scope.launch {
                        val result = createWalletOnServer(username)
                        walletAddress = result ?: "Erro ao criar"
                    }
                },
                modifier = Modifier.padding(top = 8.dp).fillMaxWidth()
            ) {
                Text("Gerar Endereço na Blockchain")
            }

            Spacer(modifier = Modifier.height(20.dp))

            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Endereço ECO:", color = Color.Gray)
                    Text(walletAddress, style = MaterialTheme.typography.bodySmall)
                    
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    Text("Saldo Atual:", color = Color.Gray)
                    Text(balance, style = MaterialTheme.typography.headlineSmall, color = Color(0xFF4CAF50))
                }
            }

            Spacer(modifier = Modifier.height(20.dp))

            Text("Módulo de Mineração (Nó)", style = MaterialTheme.typography.titleMedium)
            
            Button(
                onClick = {
                    if (walletAddress.startsWith("ECO_")) {
                        isMining = true
                        scope.launch {
                            val response = requestMining(walletAddress)
                            if (response != null) {
                                val reward = response.substringAfter("\"reward\":").substringBefore(",").trim()
                                val newBalance = response.substringAfter("\"balance\":").substringBefore("}").trim()
                                miningLog = listOf("Bloco Minerado! Recompensa: +$reward ECO") + miningLog
                                balance = "$newBalance ECO"
                            } else {
                                miningLog = listOf("Falha na comunicação com o Nó.") + miningLog
                            }
                            isMining = false
                        }
                    }
                },
                enabled = walletAddress.startsWith("ECO_") && !isMining,
                colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFFF9800)),
                modifier = Modifier.padding(top = 8.dp).fillMaxWidth()
            ) {
                if (isMining) {
                    CircularProgressIndicator(color = Color.White, modifier = Modifier.size(20.dp))
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Nó Executando Proof of Work...")
                } else {
                    Text("Solicitar Mineração de Bloco")
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            Text("Enviar Moedas ECO (Taxa de 1%)", style = MaterialTheme.typography.titleMedium, color = Color(0xFF2196F3))
            Spacer(modifier = Modifier.height(8.dp))

            TextField(
                value = destAddress,
                onValueChange = { destAddress = it },
                label = { Text("Endereço de Destino (ECO_...)") },
                modifier = Modifier.fillMaxWidth()
            )
            
            Spacer(modifier = Modifier.height(8.dp))

            TextField(
                value = transferAmount,
                onValueChange = { transferAmount = it },
                label = { Text("Quantidade de ECO") },
                modifier = Modifier.fillMaxWidth()
            )

            Button(
                onClick = {
                    val amountLong = transferAmount.toLongOrNull()
                    if (amountLong != null && walletAddress.startsWith("ECO_") && destAddress.isNotEmpty()) {
                        scope.launch {
                            txStatus = "Processando transferência..."
                            val newBal = executeTransfer(walletAddress, destAddress, amountLong)
                            if (newBal != null) {
                                txStatus = "Sucesso! Transação gravada na Blockchain."
                                balance = "$newBal ECO"
                                transferAmount = ""
                                destAddress = ""
                            } else {
                                txStatus = "Erro: Verifique o saldo (Valor + 1% de Taxa) ou o endereço."
                            }
                        }
                    } else {
                        txStatus = "Preencha todos os campos corretamente."
                    }
                },
                enabled = walletAddress.startsWith("ECO_"),
                colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF2196F3)),
                modifier = Modifier.padding(top = 8.dp).fillMaxWidth()
            ) {
                Text("Transferir ECO")
            }

            if (txStatus.isNotEmpty()) {
                Text(txStatus, style = MaterialTheme.typography.bodySmall, color = if(txStatus.startsWith("Sucesso")) Color(0xFF4CAF50) else Color.Red, modifier = Modifier.padding(top = 8.dp))
            }

            Spacer(modifier = Modifier.height(20.dp))
            Text("Histórico do Nó Local:", style = MaterialTheme.typography.bodyMedium, color = Color.Gray)
        }

        items(miningLog) { log ->
            Text("• $log", style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(vertical = 2.dp))
        }
    }
}

suspend fun createWalletOnServer(user: String): String? = withContext(Dispatchers.IO) {
    try {
        val url = URL("http://10.0.2.2:8080/wallet/create")
        val conn = url.openConnection() as HttpURLConnection
        conn.requestMethod = "POST"
        conn.doOutput = true
        conn.setRequestProperty("Content-Type", "application/json")

        val jsonInputString = "{\"username\": \"$user\"}"
        conn.outputStream.use { it.write(jsonInputString.toByteArray()) }

        val response = conn.inputStream.bufferedReader().readText()
        response.substringAfter("\"address\":\"").substringBefore("\"")
    } catch (e: Exception) {
        null
    }
}

suspend fun requestMining(address: String): String? = withContext(Dispatchers.IO) {
    try {
        val url = URL("http://10.0.2.2:8080/mine?address=$address")
        val conn = url.openConnection() as HttpURLConnection
        conn.requestMethod = "GET"
        if (conn.responseCode == 200) {
            conn.inputStream.bufferedReader().readText()
        } else null
    } catch (e: Exception) {
        null
    }
}

suspend fun executeTransfer(from: String, to: String, amount: Long): String? = withContext(Dispatchers.IO) {
    try {
        val url = URL("http://10.0.2.2:8080/transfer")
        val conn = url.openConnection() as HttpURLConnection
        conn.requestMethod = "POST"
        conn.doOutput = true
        conn.setRequestProperty("Content-Type", "application/json")

        val jsonInputString = "{\"from\": \"$from\", \"to\": \"$to\", \"amount\": $amount}"
        conn.outputStream.use { it.write(jsonInputString.toByteArray()) }

        if (conn.responseCode == 200) {
            val response = conn.inputStream.bufferedReader().readText()
            response.substringAfter("\"new_balance\":").substringBefore(",")
        } else {
            null
        }
    } catch (e: Exception) {
        null
    }
}

@Composable
fun EcopactoWalletTheme(content: @Composable () -> Unit) {
    MaterialTheme(content = content)
}
