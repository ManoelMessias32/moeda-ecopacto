package com.ecopacto.moeda

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL

// Paleta de cores Ecopacto v2
val EcopactoBg = Color(0xFF0B0E14)
val EcopactoCard = Color(0xFF151921)
val EcopactoPrimary = Color(0xFF22C55E)
val EcopactoSecondary = Color(0xFF3B82F6)
val EcopactoText = Color(0xFFF8FAFC)
val EcopactoMuted = Color(0xFF94A3B8)

class WalletActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            EcopactoTheme {
                WalletScreen()
            }
        }
    }
}

@Composable
fun WalletScreen() {
    val scope = rememberCoroutineScope()
    var walletAddress by remember { mutableStateOf("ECO_...") }
    var balance by remember { mutableStateOf("0") }
    var destAddress by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var statusMsg by remember { mutableStateOf("") }
    var showSendForm by remember { mutableStateOf(false) }

    Surface(modifier = Modifier.fillMaxSize(), color = EcopactoBg) {
        Column(modifier = Modifier.padding(24.dp)) {
            // Header
            Text(
                text = "ECOPACTO v2",
                color = EcopactoText,
                fontSize = 24.sp,
                fontWeight = FontWeight.ExtraBold,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )

            Spacer(modifier = Modifier.height(40.dp))

            // Balance Card
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Text("Saldo Disponível", color = EcopactoMuted, fontSize = 14.sp)
                Text(
                    text = "$balance ECO",
                    color = EcopactoPrimary,
                    fontSize = 48.sp,
                    fontWeight = FontWeight.Bold
                )
            }

            Spacer(modifier = Modifier.height(40.dp))

            // Actions
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                Button(
                    onClick = { /* Lógica de copiar endereço */ },
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(containerColor = EcopactoCard),
                    shape = RoundedCornerShape(12.dp)
                ) {
                    Text("📥 Receber", color = EcopactoText)
                }
                Button(
                    onClick = { showSendForm = !showSendForm },
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(containerColor = EcopactoPrimary),
                    shape = RoundedCornerShape(12.dp)
                ) {
                    Text("💸 Enviar", color = Color.Black, fontWeight = FontWeight.Bold)
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Address Card
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(EcopactoCard, RoundedCornerShape(16.dp))
                    .padding(16.dp)
            ) {
                Column {
                    Text("Endereço da Carteira:", color = EcopactoMuted, fontSize = 12.sp)
                    Text(
                        text = walletAddress,
                        color = EcopactoSecondary,
                        fontSize = 12.sp,
                        modifier = Modifier.padding(top = 4.dp)
                    )
                }
            }

            if (showSendForm) {
                Spacer(modifier = Modifier.height(24.dp))
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(EcopactoCard, RoundedCornerShape(16.dp))
                        .padding(16.dp)
                ) {
                    TextField(
                        value = destAddress,
                        onValueChange = { destAddress = it },
                        placeholder = { Text("Destino ECO_...", color = EcopactoMuted) },
                        modifier = Modifier.fillMaxWidth(),
                        colors = TextFieldDefaults.colors(
                            unfocusedContainerColor = Color.Transparent,
                            focusedContainerColor = Color.Transparent,
                            unfocusedIndicatorColor = EcopactoMuted,
                            focusedIndicatorColor = EcopactoPrimary
                        )
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    TextField(
                        value = amount,
                        onValueChange = { amount = it },
                        placeholder = { Text("Quantidade", color = EcopactoMuted) },
                        modifier = Modifier.fillMaxWidth(),
                        colors = TextFieldDefaults.colors(
                            unfocusedContainerColor = Color.Transparent,
                            focusedContainerColor = Color.Transparent
                        )
                    )
                    Button(
                        onClick = {
                            scope.launch {
                                val newBal = executeTransfer(walletAddress, destAddress, amount.toLongOrNull() ?: 0L)
                                if (newBal != null) {
                                    balance = newBal
                                    statusMsg = "💸 Enviado com sucesso!"
                                } else {
                                    statusMsg = "❌ Erro na transação"
                                }
                            }
                        },
                        modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = EcopactoPrimary)
                    ) {
                        Text("Confirmar Envio (Taxa 1%)", color = Color.Black)
                    }
                }
            }
            
            Text(
                text = statusMsg,
                color = EcopactoSecondary,
                fontSize = 12.sp,
                modifier = Modifier.padding(top = 16.dp).align(Alignment.CenterHorizontally)
            )
        }
    }
}

const val PROD_URL = "https://ecopacto-api-production.up.railway.app"

suspend fun executeTransfer(from: String, to: String, amount: Long): String? = withContext(Dispatchers.IO) {
    try {
        val url = URL("$PROD_URL/transfer")
        val conn = url.openConnection() as HttpURLConnection
        conn.requestMethod = "POST"
        conn.doOutput = true
        conn.setRequestProperty("Content-Type", "application/json")
        val jsonInputString = "{\"from\": \"$from\", \"to\": \"$to\", \"amount\": $amount}"
        conn.outputStream.use { it.write(jsonInputString.toByteArray()) }
        if (conn.responseCode == 200) {
            val response = conn.inputStream.bufferedReader().readText()
            response.substringAfter("\"new_balance\":").substringBefore(",")
        } else null
    } catch (e: Exception) { null }
}

@Composable
fun EcopactoTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = darkColorScheme(
            primary = EcopactoPrimary,
            background = EcopactoBg,
            surface = EcopactoCard
        ),
        content = content
    )
}
