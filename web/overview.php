<html>
<head>
<script src="http://ajax.googleapis.com/ajax/libs/jquery/1.11.2/jquery.min.js"></script>
<script>
$(document).ready(function(){
    $("#flip").click(function(){
        $("#panel").slideToggle("fast");
    });
});

function setDays(d)
{
	document.getElementsByName('displayDays')[0].value=d;
	document.getElementsByName('days')[0].value=d;
}

function recomputeBitfield()
{
	var currentBitfield = 0;

	for(var thisSelector=document.getElementsByName('domainSelector').length; thisSelector--;)
	{
		var options = document.getElementsByName('domainSelector')[thisSelector].getElementsByTagName('option');
		var selector_value = 0;

		for(var i=options.length; i--;)
		{
			if(options[i].selected)
			{
				// if we've selected the "don't care" option, then any values we've found so far for this selector don't matter
				if(options[i].value == 0)
				{
					selector_value = 0;
					break;
				}
				selector_value = selector_value | options[i].value;
			}
		}

		if(selector_value != 0)
		{
			if(currentBitfield != 0)
			{
				currentBitfield = currentBitfield & selector_value;
			}
			else
			{
				currentBitfield = selector_value;
			}
		}
	}

	document.getElementsByName('domains')[0].value=currentBitfield;
}

</script>

<style>
#panel, #flip {
    padding: 5px;
    text-align: center;
    background-color: #ffffff;
}

#panel {
    padding: 50px;
    display: none;
}

#spacingtd {
    border-left: thin double #000000;
}
</style>
</head>
<body>

<?php
$days = 1;

if(isset($_GET["days"]))
{
	if(is_numeric($_GET["days"]))
	{
		if($_GET["days"] > 0)
		{
			$days = $_GET["days"];
		}
	}
}

$requested_domain_bitfield = 0;
if(isset($_GET["domains"]))
{
	if(is_numeric($_GET["domains"]))
	{
		if($_GET["domains"] > 0)
		{
			$requested_domain_bitfield = $_GET["domains"];
		}
	}
}

include 'database.php';

// how many days ago did our current sprint start?
// '2013-12-28' is the day one sprint started, and they restart every 3 weeks
$query = "SELECT extract(days from current_date-(select date '2013-12-28' + interval '3 weeks' * (((extract(epoch from current_date)-extract(epoch from date '2013-12-28'))::int / 1728000)-1) ))";
$daysInSprint = pg_query($query) or die('Query failed: ' . pg_last_error());

// how far back are we going to be looking for events?
// We need an actual value here so that when we use it in a where clause, that value gets pushed to the foriegn servers
$query = "select now()-interval '1 day'*$days";
$result = pg_query($query) or die('Query failed: ' . pg_last_error());
$timespan = pg_fetch_result($result,0,0);


// the various domains we have to iterate over
$query = "select distinct skeys(domains) from data_sources";
$domainKeys = pg_query($query) or die('Query failed: ' . pg_last_error());

// the data sources we'll build a bitfield for
$query = "select 2^(id-1),schema,name from data_sources order by id";
$dataSourceBitfields = pg_query($query) or die('Query failed: ' . pg_last_error());

// compute the bitfield that includes every data source
$query = "select bit_or((2^(id-1))::int) from data_sources";
$result = pg_query($query) or die('query failed: ' . pg_last_error());
$all_domains = pg_fetch_result($result,0,0);

// Deconstruct our bitfield to determine which datasources to pull from
$selected_datasources = array();
while($ds = pg_fetch_row($dataSourceBitfields))
{
	// if we haven't requested any data source explicitly, or we have and it includes this one, then add this datasource to our list
	if ($requested_domain_bitfield == 0 || (intval($ds[0]) & intval($requested_domain_bitfield)))
	{
		$selected_datasources[] = array($ds[1],$ds[2]);
	}
}
$events = "select bucket_id,finished from \"" . $selected_datasources[0][0] . "\".events where finished > '$timespan'";
for($i=1;$i < count($selected_datasources); $i++)
{
	$events = $events . " union all select bucket_id,finished from \"" . $selected_datasources[$i][0] . "\".events where finished > '$timespan'";
}

function second_value($a)
{
        return $a[1];
}

$pretty_datasources = "everywhere";
if($requested_domain_bitfield > 0 && $requested_domain_bitfield != $all_domains)
{
	$pretty_datasources = implode(", ", array_map("second_value",$selected_datasources));
}

echo "<div id=\"flip\">\n";
echo "<h2>Showing log counts over the last $days days from $pretty_datasources</h2>\n";
echo "show instead....</div>\n";

echo "<div id=\"panel\">";
echo "<table width=100% cellpadding=10><tr><td colspan=2 align=\"middle\"><b>When</b></td><td id=\"spacingtd\"></td>";
while ($row = pg_fetch_row($domainKeys))
{
	echo "<td><b>$row[0]</b></td>";
}

echo "</tr>\n";
echo "<tr><td>";
echo "<button onclick=\"setDays(.042)\">1 hour</button><br>";
echo "<button onclick=\"setDays(1)\">1 day</button><br>";
echo "<button onclick=\"setDays(7)\">1 week</button><br>";
echo "<button onclick=\"setDays(", pg_fetch_result($daysInSprint,0,0),")\">this sprint</button><br>";
echo "<button onclick=\"setDays(90)\">1 quarter</button><br>";
echo "</td><td valign=\"middle\"><input name=\"displayDays\" type=\"number\" value=\"$days\"></td><td id=\"spacingtd\"></td>";

pg_result_seek($domainKeys,0);
while ($row = pg_fetch_row($domainKeys))
{
	echo "<td><select name=\"domainSelector\" multiple onchange=\"recomputeBitfield();\"><option value=\"0\">Don't care</option>";
	$query = "select bit_or((2^(id-1))::int),domains->'$row[0]' from data_sources where defined(domains,'$row[0]') group by domains->'$row[0]'";
	$result = pg_query($query) or die('Query failed: ' . pg_last_error());
	while($value = pg_fetch_row($result))
	{
		$selected = "";
		// if we have not explicitly requested any data sources, or we have and this domain has a datasource in that bitfield, then this domain item should be selected
		if ($requested_domain_bitfield == 0 || (intval($requested_domain_bitfield) & intval($value[0])))
		{
			$selected = "selected";
		}
		echo "<option $selected value=\"", $value[0], "\">", $value[1], "</option>";
	}
	echo "</select></td>";
}

echo "</tr></table><hr><table width=100%><tr><td align=\"middle\"><form>";
echo "<input type=\"hidden\" name=\"days\" value=\"$days\">";
if($requested_domain_bitfield == 0)
{
	$requested_domain_bitfield = $all_domains;
}
echo "<input type=\"hidden\" name=\"domains\" value=\"$requested_domain_bitfield\">";
echo "<button>Do it</button></form></td></tr></table></div>";

// Join all events against the buckets in a data source. All data sources should have the same content in buckets, 
// so it doesn't matter which we choose... given that we'll always have at least one, just do the first one.
$query = "with all_events as ($events) SELECT count(*),name,bucket_id from all_events,\"" . $selected_datasources[0][0] . "\".buckets where bucket_id=buckets.id and buckets.active group by name,bucket_id order by name";
$namedThings = pg_query($query) or die('Query failed: ' . pg_last_error());
$query = "with all_events as ($events) SELECT count(*) from all_events where bucket_id is null";
$interestingThings = pg_query($query) or die('Query failed: ' . pg_last_error());

// Printing results in HTML
echo "<div id=\"summary\">\n";
echo "<table>\n";
    echo "\t<tr>\n";
        echo "\t\t<td>Count</td>\n";
        echo "\t\t<td>Bucket</td>\n";
    echo "\t</tr>\n";
while ($row = pg_fetch_row($namedThings))
{
    echo "\t<tr>\n";
        echo "\t\t<td><a href='bucketDetails.php?days=$days&domains=$requested_domain_bitfield&id=$row[2]'>$row[0]</a></td>\n";
	if(preg_match('/^CNVS-\d+$/',$row[1]))
	{
	        echo "\t\t<td><a href='https://instructure.atlassian.net/browse/$row[1]'>$row[1]</a></td>\n";
	}
	else
	{
	        echo "\t\t<td>" . htmlentities($row[1], ENT_QUOTES) . "</td>\n";
	}
    echo "\t</tr>\n";
}
while ($row = pg_fetch_row($interestingThings))
{
    echo "\t<tr>\n";
        echo "\t\t<td><a href='bucketDetails.php?days=$days&domains=$requested_domain_bitfield'>$row[0]</a></td>\n";
        echo "\t\t<td><b>interesting things</b></td>\n";
    echo "\t</tr>\n";
}
echo "</table>\n";

// Free resultset
pg_free_result($daysInSprint);
pg_free_result($domainKeys);
pg_free_result($dataSourceBitfields);
pg_free_result($namedThings);
pg_free_result($interestingThings);


// Closing connection
pg_close($dbconn);
?>
</div>
</body>
</html>
